package api

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"io/ioutil"
	"mime"
	"net/http"
	"net/http/httputil"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/common/expfmt"

	"github.com/gogo/protobuf/proto"
	"github.com/golang/snappy"
	"github.com/matttproud/golang_protobuf_extensions/pbutil"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/pkg/labels"
	"github.com/prometheus/prometheus/pkg/textparse"
	"github.com/prometheus/prometheus/pkg/timestamp"
	"github.com/prometheus/prometheus/prompb"
	"go.uber.org/zap"
)

func init() {
	_ = prometheus.Register(numOfSendErrors)
	_ = prometheus.Register(numOfSendSuccess)
}

var numOfSendErrors = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "numOfSendErrors",
		Help: "count send error by type",
	},
	[]string{"type"},
)

var numOfSendSuccess = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "numOfSendSuccess",
		Help: "count send success by type",
	},
	[]string{"type"},
)

// ServePromWrite handles prometheus remote write requests.
func (s *Service) ServePromWrite(c *gin.Context) {
	// Take will sleep until you can continue.
	s.limiter.Take()
	if c.Request.ContentLength > 0 {
		if s.bodySizeLimit > 0 && c.Request.ContentLength > int64(s.bodySizeLimit) {
			http.Error(c.Writer, http.StatusText(http.StatusRequestEntityTooLarge), http.StatusRequestEntityTooLarge)
			return
		}
	}

	data, err := ioutil.ReadAll(c.Request.Body)
	if err != nil {
		http.Error(c.Writer, err.Error(), http.StatusInternalServerError)
		return
	}
	if len(data) > s.bodySizeLimit {
		http.Error(c.Writer, "request entity too large", http.StatusRequestEntityTooLarge)
		return
	}
	if err := s.queue.Push(data); err != nil {
		http.Error(c.Writer, err.Error(), http.StatusInternalServerError)
		return
	}
}

// Push implements pushgateway handler.
func (s *Service) Push(jobBase64Encoded bool) func(c *gin.Context) { //nolint: gocognit
	h := func(c *gin.Context) {
		if !s.pushGatewayEnable {
			http.Error(c.Writer, "pushGateway mode not enabled", http.StatusInternalServerError)
			return
		}

		s.limiter.Take()
		if c.Request.ContentLength > 0 {
			if s.bodySizeLimit > 0 && c.Request.ContentLength > int64(s.bodySizeLimit) {
				http.Error(c.Writer,
					http.StatusText(http.StatusRequestEntityTooLarge), http.StatusRequestEntityTooLarge)
				return
			}
		}

		job := c.Param("job")
		if jobBase64Encoded {
			var err error
			if job, err = decodeBase64(job); err != nil {
				http.Error(c.Writer, fmt.Sprintf("invalid base64 encoding in job name %q: %v", job, err), http.StatusBadRequest)
				s.logger.Error("invalid base64 encoding in job name", zap.String("jon", job), zap.Error(err))
				return
			}
		}
		labelsString := c.Param("labels")
		labelss, err := splitLabels(labelsString)
		if err != nil {
			http.Error(c.Writer, err.Error(), http.StatusBadRequest)
			s.logger.Error("failed to parse URL", zap.String("url", labelsString), zap.Error(err))
			return
		}
		if job == "" {
			http.Error(c.Writer, "job name is required", http.StatusBadRequest)
			s.logger.Error("job name is required")
			return
		}
		labelss["job"] = job

		var metricFamilies map[string]*dto.MetricFamily
		ctMediatype, ctParams, ctErr := mime.ParseMediaType(c.Request.Header.Get("Content-Type"))
		if ctErr == nil && ctMediatype == "application/vnd.google.protobuf" &&
			ctParams["encoding"] == "delimited" &&
			ctParams["proto"] == "io.prometheus.client.MetricFamily" {
			metricFamilies = map[string]*dto.MetricFamily{}
			for {
				mf := &dto.MetricFamily{}
				if _, err = pbutil.ReadDelimited(c.Request.Body, mf); err != nil {
					if err == io.EOF {
						err = nil
					}
					break
				}
				metricFamilies[mf.GetName()] = mf
			}
		} else {
			// We could do further content-type checks here, but the
			// fallback for now will anyway be the text format
			// version 0.0.4, so just go for it and see if it works.
			var parser expfmt.TextParser
			metricFamilies, err = parser.TextToMetricFamilies(c.Request.Body)
		}
		if err != nil {
			http.Error(c.Writer, err.Error(), http.StatusBadRequest)
			s.logger.Error("failed to parse text", zap.Error(err))
			return
		}

		wb := new(bytes.Buffer)
		for _, metric := range metricFamilies {
			if _, err := expfmt.MetricFamilyToText(wb, metric); err != nil {
				s.logger.Error("MetricFamilyToText error", zap.Error(err))
				continue
			}
		}

		if err := s.rePackage(labelss, wb.Bytes()); err != nil {
			s.logger.Error("repackage error", zap.Error(err))
			http.Error(c.Writer, err.Error(), http.StatusInternalServerError)
			return
		}
		c.Writer.WriteHeader(http.StatusAccepted)
	}
	return h
}

func (s *Service) rePackage(gk map[string]string, data []byte) error {
	//TODO: support openmetrics
	p := textparse.NewPromParser(data)
	t := timestamp.FromTime(time.Now())
	var wq prompb.WriteRequest
	for {
		var et textparse.Entry
		var err error
		if et, err = p.Next(); err != nil {
			if err == io.EOF {
				err = nil //nolint: ineffassign
			}
			break
		}
		switch et {
		case textparse.EntryType:
			continue
		case textparse.EntryHelp:
			continue
		case textparse.EntryUnit:
			continue
		case textparse.EntryComment:
			continue
		default:
		}
		var ts prompb.TimeSeries
		_, tp, v := p.Series()
		if tp != nil {
			t = *tp
		}

		var lset labels.Labels
		p.Metric(&lset)

		// The label set may be set to nil to indicate dropping.
		if lset == nil {
			s.logger.Error("found empty label set, drop it")
			continue
		}
		for _, labelPair := range lset {
			var l prompb.Label
			l.Name = labelPair.Name
			l.Value = labelPair.Value
			ts.Labels = append(ts.Labels, &l)
		}

		// add group key labels
		for k, v := range gk {
			ts.Labels = append(ts.Labels, &prompb.Label{Name: k, Value: v})
		}

		ts.Samples = append(ts.Samples, prompb.Sample{Value: v, Timestamp: t})
		wq.Timeseries = append(wq.Timeseries, &ts)
	}

	data, err := proto.Marshal(&wq)
	if err != nil {
		return err
	}
	res := snappy.Encode(nil, data)
	if err := s.queue.Push(res); err != nil {
		return err
	}
	return nil
}

// decodeBase64 decodes the provided string using the “Base 64 Encoding with URL
// and Filename Safe Alphabet” (RFC 4648). Padding characters (i.e. trailing
// '=') are ignored.
func decodeBase64(s string) (string, error) {
	b, err := base64.RawURLEncoding.DecodeString(strings.TrimRight(s, "="))
	return string(b), err
}

// splitLabels splits a labels string into a label map mapping names to values.
func splitLabels(labels string) (map[string]string, error) {
	result := map[string]string{}
	if len(labels) <= 1 {
		return result, nil
	}
	components := strings.Split(labels[1:], "/")
	if len(components)%2 != 0 {
		return nil, fmt.Errorf("odd number of components in label string %q", labels)
	}

	for i := 0; i < len(components)-1; i += 2 {
		name, value := components[i], components[i+1]
		trimmedName := strings.TrimSuffix(name, Base64Suffix)
		if !model.LabelNameRE.MatchString(trimmedName) ||
			strings.HasPrefix(trimmedName, model.ReservedLabelPrefix) {
			return nil, fmt.Errorf("improper label name %q", trimmedName)
		}
		if name == trimmedName {
			result[name] = value
			continue
		}
		decodedValue, err := decodeBase64(value)
		if err != nil {
			return nil, fmt.Errorf("invalid base64 encoding for label %s=%q: %v", trimmedName, value, err)
		}
		result[trimmedName] = decodedValue
	}
	return result, nil
}

// Delete implements pushgateway delete handler
func (s *Service) Delete(jobBase64Encoded bool) func(c *gin.Context) {
	h := func(c *gin.Context) {

	}
	return h
}

func (s *Service) ProxyQuery(c *gin.Context) {
	if !s.queryEnable {
		http.Error(c.Writer, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}

	director := func(req *http.Request) {
		req.Header.Add("X-Forwarded-Host", req.Host)
		req.Header.Add("X-Origin-Host", s.queryAddr)
		req.URL.Scheme = "http"
		req.URL.Host = s.queryAddr
	}

	proxy := &httputil.ReverseProxy{Director: director}
	proxy.ServeHTTP(c.Writer, c.Request)
}

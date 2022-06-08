package backend

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/golang/snappy"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/prometheus/prompb"
	"go.uber.org/zap"
)

var (
	EndpointSendFailed = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "endpoint_send_failed",
			Help:      "The failed number of samples sended.",
		},
		[]string{"endpoint", "type"},
	)
	EndpointSendSuccess = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "endpoint_send_success",
			Help:      "The success number of samples sended.",
		},
		[]string{"endpoint"},
	)
	EndpointSendDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "endpoint_send_duration_seconds",
			Help:    "The HTTP request to prometheus store latencies in seconds.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"code", "method", "backend"},
	)
)

// default time series batch send number.
var defaultBatchSend = 100

// default flush samples duration.
var flushSamplesDuration = 60 * time.Second

// Endpoint interface.
type Endpoint interface {
	// Start endpoint send data.
	Start()
	// Stop endpoint.
	Stop()
	// Send samples to endpoints.
	Send([]*prompb.Label, []prompb.Sample) error
	// Addr returns endpoint's address.
	Addr() string
}

// HTTPEndpoint implements an HTTP endpoint.
type HTTPEndpoint struct {
	client      *http.Client
	addr        string
	cache       chan prompb.TimeSeries
	concurrency int
	logger      *zap.Logger
	done        chan struct{}
}

// NewHTTPEndpoint creates an HTTP endpoint.
func NewHTTPEndpoint(addr string, concurrency int, logger *zap.Logger) *HTTPEndpoint {
	if concurrency < 1 {
		concurrency = 1
	}
	return &HTTPEndpoint{
		client: &http.Client{
			Timeout: 5 * time.Second,
			Transport: &http.Transport{
				MaxIdleConnsPerHost: 30,
				IdleConnTimeout:     10 * time.Minute,
			},
		},
		addr:        addr,
		concurrency: concurrency,
		cache:       make(chan prompb.TimeSeries, defaultBatchSend*2),
		done:        make(chan struct{}),
		logger:      logger.With(zap.String("service", "endpoint")),
	}
}

func (e *HTTPEndpoint) rollback(data []*prompb.TimeSeries) {
	for _, ts := range data {
		e.cache <- *ts
	}
}

// Start starts the task.
func (e *HTTPEndpoint) Start() {
	e.logger.Info("start endpoint", zap.String("addr", e.addr))
	var timeSeriesData []*prompb.TimeSeries
	limiter := NewLimit(e.concurrency)
	ticker := time.NewTicker(flushSamplesDuration)
	for {
		select {
		case <-e.done:
			return
		case m := <-e.cache:
			timeSeriesData = append(timeSeriesData, &m)
			if len(timeSeriesData) < defaultBatchSend {
				continue
			}
			limiter.Take()
			tmp := make([]*prompb.TimeSeries, len(timeSeriesData))
			copy(tmp, timeSeriesData)
			go func([]*prompb.TimeSeries) {
				defer limiter.Release()
				e.doSend(tmp)
			}(tmp)
			timeSeriesData = []*prompb.TimeSeries{}
		case t := <-ticker.C:
			e.logger.Info("flush samples by ticker", zap.String("ticker", t.String()))
			if len(timeSeriesData) == 0 {
				continue
			}
			limiter.Take()
			tmp := make([]*prompb.TimeSeries, len(timeSeriesData))
			copy(tmp, timeSeriesData)
			go func([]*prompb.TimeSeries) {
				defer limiter.Release()
				e.doSend(tmp)
			}(tmp)
			timeSeriesData = []*prompb.TimeSeries{}
		}
	}
}

func (e *HTTPEndpoint) doSend(tmp []*prompb.TimeSeries) {
	client := e.client

	e.logger.Info("send to endpoint", zap.String("endpoint", e.addr), zap.Int("size", len(tmp)))
	var wq prompb.WriteRequest
	wq.Timeseries = tmp
	data, err := proto.Marshal(&wq)
	if err != nil {
		EndpointSendFailed.WithLabelValues(e.addr, "protoMarshalFailed").Inc()
		e.logger.Error("endpoint batch send", zap.Error(err))
		return
	}
	res := snappy.Encode(nil, data)
	url := fmt.Sprintf("%s/api/v1/write", e.addr)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(res))
	if err != nil {
		EndpointSendFailed.WithLabelValues(e.addr, "newRequestFailed").Inc()
		e.logger.Error("batch send new req", zap.Error(err))
		go e.rollback(tmp)
		return
	}
	start := time.Now()
	resp, err := client.Do(req)
	if err != nil {
		EndpointSendFailed.WithLabelValues(e.addr, "httpClientFailed").Inc()
		e.logger.Error("endpoint batch send", zap.Error(err))
		time.Sleep(1 * time.Second)
		go e.rollback(tmp)
		return
	}
	defer func() {
		_, _ = io.Copy(ioutil.Discard, resp.Body) // Avoid resource leak.
		resp.Body.Close()
	}()
	if resp.StatusCode/100 != 2 {
		body, _ := ioutil.ReadAll(resp.Body)
		EndpointSendFailed.WithLabelValues(e.addr, "httpStatusFailed").Inc()
		e.logger.Error("status code not OK", zap.Int("code", resp.StatusCode), zap.String("body", string(body)))
		return
	}
	e.logger.Info("success consum message size:", zap.Int("size", len(data)))
	EndpointSendSuccess.WithLabelValues(e.addr).Inc()
	elapsed := time.Since(start).Seconds()
	EndpointSendDuration.WithLabelValues(strconv.Itoa(resp.StatusCode), req.Method, e.addr).Observe(elapsed)
}

// Stop stops the task.
func (e *HTTPEndpoint) Stop() {
	e.logger.Info("exit endpoint", zap.String("addr", e.addr))
	close(e.done)
}

// Addr returns the endpoint's address.
func (e *HTTPEndpoint) Addr() string {
	return e.addr
}

// Send sends samples to endpoint.
func (e *HTTPEndpoint) Send(l []*prompb.Label, s []prompb.Sample) error {
	var timeseries prompb.TimeSeries
	timeseries.Labels = l
	timeseries.Samples = s
	select {
	case e.cache <- timeseries:
		return nil
	case <-e.done:
		e.logger.Info("endpoint closed, exit Send")
		return nil
	}
}

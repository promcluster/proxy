package consumer

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/promcluster/proxy/pkg/backend"
	"github.com/promcluster/proxy/pkg/filter"

	"github.com/gogo/protobuf/proto"
	"github.com/golang/snappy"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/prompb"
	"go.uber.org/zap"
)

var namespace = "promcluster-proxy"
var subsystem = "consumer"

// default copies number
var defaultReplicationFactor = 1

var (
	consumeMessageFailed = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "cosume_message_failed",
			Help:      "The failed number of message consumer handled.",
		},
		[]string{"type"},
	)
	consumeMessageSuccess = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "cosume_message_success",
			Help:      "The success number of message consumer handled.",
		},
		[]string{"endpoint"},
	)
)

// RemoteConsumer writes data to remote.
type RemoteConsumer struct {
	backend backend.Backend
	filters []filter.Filter

	logger     *zap.Logger
	registerer prometheus.Registerer
}

// NewRemoteConsumer creates a new consumer.
func NewRemoteConsumer(
	ctx context.Context,
	reg prometheus.Registerer,
	b backend.Backend,
	fs []filter.Filter,
	l *zap.Logger) *RemoteConsumer {
	reg.MustRegister(consumeMessageFailed, consumeMessageSuccess)
	return &RemoteConsumer{
		backend:    b,
		registerer: reg,
		filters:    fs,
		logger:     l,
	}
}

// HandleMessage implements Consumer interface.
// if return true, workers will retry.
func (r *RemoteConsumer) HandleMessage(msg []byte) (bool, error) {
	reqBuf, err := snappy.Decode(nil, msg)
	if err != nil {
		r.logger.Error("consumer snappy decode", zap.Error(err))
		// drop this bad format message
		consumeMessageFailed.WithLabelValues("snappyDecode").Inc()
		return false, err
	}

	var req prompb.WriteRequest
	if err = proto.Unmarshal(reqBuf, &req); err != nil {
		r.logger.Error("consumer proto decode", zap.Error(err))
		// drop this bad format message
		consumeMessageFailed.WithLabelValues("protoDecode").Inc()
		return false, err
	}

	if len(req.Timeseries) < 1 {
		r.logger.Error("bad data length", zap.Int("size", len(req.Timeseries)))
		// drop this bad format message
		consumeMessageFailed.WithLabelValues("emptyTimeseries").Inc()
		return false, errors.New("empty timeseries")
	}

NEXT:
	for _, ts := range req.Timeseries {
		lbs := ts.GetLabels()
		lset := make(model.LabelSet)
		for _, l := range lbs {
			lset[model.LabelName(l.Name)] = model.LabelValue(l.Value)
		}

		// check timestamp vaild
		for _, sample := range ts.Samples {
			if sample.Timestamp/1000-time.Now().Unix() > 60 {
				r.logger.Error(
					"found out of order sample(a)",
					zap.Int64("ts", sample.Timestamp),
					zap.String("labels",
						lset.String()))
				return false, fmt.Errorf("found out of order sample")
			}
		}

		// filters handlers
		for _, f := range r.filters {
			err = f.Filt(&lset)
			if err != nil {
				r.logger.Info("filter", zap.Error(err))
				continue NEXT
			}
		}

		// Don't use json encode labels.
		// Labels need to be sorted.
		endpoints, err := r.backend.Endpoints(filter.LabelsString(&lset), defaultReplicationFactor)
		if err != nil {
			r.logger.Error("get endpoints from backend", zap.Error(err))
			consumeMessageFailed.WithLabelValues("getEndpoints").Inc()
			return true, err
		}

		for _, e := range endpoints {
			r.logger.Debug(
				"Samples Detail",
				zap.Any("Labels", lbs),
				zap.Any("Samples", ts.Samples),
				zap.String("Endpoint", e.Addr()),
			)
			err = e.Send(lbs, ts.Samples)
			if err != nil {
				r.logger.Error("send to endpoints", zap.Error(err))
				consumeMessageFailed.WithLabelValues("sendEndpoints").Inc()
				continue
			}
			consumeMessageSuccess.WithLabelValues(e.Addr()).Inc()
		}
	}
	return false, nil
}

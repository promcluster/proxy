package consumer

import (
	"context"
	"testing"
	"time"

	"github.com/promcluster/proxy/pkg/backend"
	"github.com/promcluster/proxy/pkg/filter"

	"github.com/gogo/protobuf/proto"
	"github.com/golang/snappy"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/prometheus/prompb"
	"go.uber.org/zap"
)

func TestRemoteConsumer(t *testing.T) {
	var wq prompb.WriteRequest
	var s prompb.Sample
	s.Value = 3.14
	s.Timestamp = time.Now().Unix() * 1000
	var ts prompb.TimeSeries
	ts.Samples = []prompb.Sample{s}
	ts.Labels = []*prompb.Label{&prompb.Label{Name: "lname", Value: "v1"}}
	wq.Timeseries = []*prompb.TimeSeries{&ts}

	data, err := proto.Marshal(&wq)
	if err != nil {
		t.Fatal(err)
	}
	res := snappy.Encode(nil, data)
	m := &mockBackend{}
	f := filter.NewEmptyFilter()
	r := NewRemoteConsumer(context.TODO(), prometheus.DefaultRegisterer, m, []filter.Filter{f}, zap.NewExample())
	_, err = r.HandleMessage(res)
	if err != nil {
		t.Fatal(err)
	}
}

type mockBackend struct{}

func (m *mockBackend) Endpoints(key string, rep int) ([]backend.Endpoint, error) {
	return []backend.Endpoint{&mockEndpoint{}}, nil
}

type mockEndpoint struct{}

func (e *mockEndpoint) Start() {}

func (e *mockEndpoint) Stop() {}

func (e *mockEndpoint) Addr() string { return "test" }

func (e *mockEndpoint) Send([]*prompb.Label, []prompb.Sample) error {
	return nil
}

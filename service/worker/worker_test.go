package worker

import (
	"context"
	"testing"

	"github.com/promcluster/proxy/pkg/consumer"
	"github.com/promcluster/proxy/pkg/queue"

	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
)

func TestWorker(t *testing.T) {
	l := consumer.NewLogConsumer()
	q := queue.NewChanQueue(prometheus.DefaultRegisterer, zap.NewExample())
	w := newWorker(prometheus.DefaultRegisterer, 0, q, l, zap.NewExample())

	ctx, cancel := context.WithCancel(context.Background())
	err := w.start(ctx)
	if err != nil {
		t.Fatal(err)
	}
	cancel()
}

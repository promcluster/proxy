package worker

import (
	"context"
	"testing"
	"time"

	"github.com/promcluster/proxy/pkg/consumer"
	"github.com/promcluster/proxy/pkg/queue"

	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
)

func TestMasterWorker(t *testing.T) {
	l := consumer.NewLogConsumer()
	q := queue.NewChanQueue(prometheus.DefaultRegisterer, zap.NewExample())
	ctx, cancel := context.WithCancel(context.Background())
	err := StartWorkers(ctx, prometheus.DefaultRegisterer, 5, q, l, zap.NewExample())

	if err != nil {
		t.Fatal(err)
	}
	cancel()
	time.Sleep(2 * time.Second)
}

package worker

import (
	"context"
	"errors"

	pkgc "github.com/promcluster/proxy/pkg/consumer"
	pkgq "github.com/promcluster/proxy/pkg/queue"

	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
)

// StartWorkers starts workers in background.
func StartWorkers(
	ctx context.Context,
	reg prometheus.Registerer,
	num int,
	q pkgq.Queue,
	c pkgc.Consumer,
	l *zap.Logger) error {
	if num < 0 {
		return errors.New("worker num must greater than 0")
	}

	for i := 1; i <= num; i++ {
		w := newWorker(reg, i, q, c, l)
		if err := w.start(ctx); err != nil {
			return err
		}
	}
	return nil
}

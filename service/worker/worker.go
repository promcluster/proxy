package worker

import (
	"context"
	"strings"

	pkgc "github.com/promcluster/proxy/pkg/consumer"
	pkgq "github.com/promcluster/proxy/pkg/queue"

	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
)

type worker struct {
	id       int
	queue    pkgq.Queue
	consumer pkgc.Consumer

	registerer prometheus.Registerer
	logger     *zap.Logger
}

func newWorker(reg prometheus.Registerer, id int, q pkgq.Queue, c pkgc.Consumer, l *zap.Logger) *worker {
	return &worker{
		id:         id,
		queue:      q,
		consumer:   c,
		registerer: reg,
		logger:     l.With(zap.String("service", "worker")),
	}
}

func (w *worker) start(ctx context.Context) error {
	loop := func(ctx context.Context, w *worker) {
		for {
			select {
			case <-ctx.Done():
				w.logger.Info("worker stopped", zap.Int("ID", w.id))
				return
			default:
				m, err := w.queue.Pop()
				if err != nil {
					w.logger.Error("pop message", zap.Error(err))
					continue
				}
				needRetry, err := w.consumer.HandleMessage(m)
				if err != nil {
					// we can not push these data back into queue.
					if strings.Contains(err.Error(), "out of bounds") ||
						strings.Contains(err.Error(), "out of order sample") ||
						strings.Contains(err.Error(), "duplicate sample for timestamp") {
						w.logger.Error("drop message", zap.Error(err))
						continue
					}

					w.logger.Error("handle message", zap.Error(err))
					if needRetry {
						if err := w.queue.Push(m); err != nil {
							w.logger.Error("message requeue", zap.Error(err))
						}
					}
				}
			}
		}
	}

	go loop(ctx, w)
	w.logger.Info("worker started", zap.Int("ID", w.id))
	return nil
}

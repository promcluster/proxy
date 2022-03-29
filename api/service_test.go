package api

import (
	"context"
	"testing"

	"github.com/promcluster/proxy/config"
	pkgq "github.com/promcluster/proxy/pkg/queue"

	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/ratelimit"
	"go.uber.org/zap"
)

func TestService(t *testing.T) {
	queue := pkgq.NewChanQueue(prometheus.DefaultRegisterer, zap.NewExample())

	s, err := New(
		prometheus.DefaultRegisterer,
		config.APIConfiguration{Listen: ":9994"},
		queue, ratelimit.NewUnlimited(), zap.NewExample())
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Start(context.TODO()); err != nil {
		t.Fatal(err)
	}

	if err := s.Close(context.TODO()); err != nil {
		t.Fatal(err)
	}
}

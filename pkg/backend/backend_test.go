package backend

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
)

func TestBackend(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	b := NewPromServer(ctx, prometheus.DefaultRegisterer, "dns+qq.com:80", 2, 1*time.Second, zap.NewExample())
	time.Sleep(2 * time.Second)
	es, err := b.Endpoints("test", 1)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(es)
	cancel()
}

package backend

import (
	"go.uber.org/zap"
	"testing"
)

func TestEndpoint(t *testing.T) {
	e := NewHTTPEndpoint("http://127.0.0.1", 1, zap.NewExample())
	if e.Addr() != "http://127.0.0.1" {
		t.Fatal("get bad address")
	}
	go e.Start()
	e.Stop()
}

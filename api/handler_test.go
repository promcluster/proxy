package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/promcluster/proxy/config"
	pkgq "github.com/promcluster/proxy/pkg/queue"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/ratelimit"
	"go.uber.org/zap"
)

func TestSendHandler(t *testing.T) {
	type Data struct {
		Content string
	}

	data := Data{"test data"}
	b, err := json.Marshal(data)
	if err != nil {
		t.Fatal(err)
	}

	req, err := http.NewRequest("POST", "localhost:80/api/v1/prom/write", bytes.NewBuffer(b))
	if err != nil {
		t.Fatal(err)
	}

	rec := httptest.NewRecorder()

	queue := pkgq.NewChanQueue(prometheus.DefaultRegisterer, zap.NewExample())
	s, err := New(
		prometheus.DefaultRegisterer,
		config.APIConfiguration{Listen: ":9990", MaxBodySizeLimit: 1024 * 1024 * 10},
		queue, ratelimit.NewUnlimited(), zap.NewExample())
	if err != nil {
		t.Fatal(err)
	}

	ctx := gin.Context{
		Request: req,
	}
	s.ServePromWrite(&ctx)
	if status := rec.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}
}

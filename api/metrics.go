package api

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	numOfSendErrors = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "numOfSendErrors",
			Help: "count send error by type",
		},
		[]string{"type"},
	)

	numOfSendSuccess = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "numOfSendSuccess",
			Help: "count send success by type",
		},
		[]string{"type"},
	)

	httpPushSize = promauto.NewSummaryVec(
		prometheus.SummaryOpts{
			Name:       "proxy_pushgateway_http_push_size_bytes",
			Help:       "HTTP request size for pushes to the Pushgateway.",
			Objectives: map[float64]float64{0.1: 0.01, 0.5: 0.05, 0.9: 0.01},
		},
		[]string{"method"},
	)
	httpPushDuration = promauto.NewSummaryVec(
		prometheus.SummaryOpts{
			Name:       "proxy_pushgateway_http_push_duration_seconds",
			Help:       "HTTP request duration for pushes to the Pushgateway.",
			Objectives: map[float64]float64{0.1: 0.01, 0.5: 0.05, 0.9: 0.01},
		},
		[]string{"method"},
	)
)

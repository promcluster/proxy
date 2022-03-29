package api

import (
	"context"
	"errors"
	"net"
	"net/http"

	"github.com/promcluster/proxy/config"
	pkgq "github.com/promcluster/proxy/pkg/queue"

	"github.com/gin-contrib/pprof"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	ginprometheus "github.com/zsais/go-gin-prometheus"
	"go.uber.org/ratelimit"
	"go.uber.org/zap"
)

const (
	// Base64Suffix is appended to a label name in the request URL path to
	// mark the following label value as base64 encoded.
	Base64Suffix = "@base64"
)

func init() {
	gin.SetMode(gin.ReleaseMode)
}

// Service provides HTTP service.
type Service struct {
	addr              string
	ln                net.Listener
	bodySizeLimit     int
	server            http.Server
	router            *gin.Engine
	queue             pkgq.Queue
	limiter           ratelimit.Limiter
	pushGatewayEnable bool

	queryEnable bool
	queryAddr   string

	registerer prometheus.Registerer
	logger     *zap.Logger
}

// New returns an uninitialized HTTP service.
func New(
	reg prometheus.Registerer,
	conf config.APIConfiguration,
	q pkgq.Queue,
	r ratelimit.Limiter,
	l *zap.Logger) (*Service, error) {
	return &Service{
		addr:              conf.Listen,
		bodySizeLimit:     conf.MaxBodySizeLimit,
		router:            gin.New(),
		server:            http.Server{},
		queue:             q,
		limiter:           r,
		pushGatewayEnable: conf.PushGatewayEnable,
		queryEnable:       conf.QueryEnable,
		queryAddr:         conf.QueryAddr,
		registerer:        reg,
		logger:            l.With(zap.String("service", "api")),
	}, nil
}

// Start the server
func (s *Service) Start(ctx context.Context) error {
	// install metrics
	p := ginprometheus.NewPrometheus("api")
	p.Use(s.router)

	// access logging
	s.router.Use(s.accessLog())

	// profile
	if config.C.API.Pprof {
		pprof.Register(s.router, "debug/pprof")
	}
	// load Authorization middleware
	if config.C.Auth.Enable {
		s.router.Use(s.auth)
	}

	// init routes
	s.initHandler()
	s.server.Handler = s.router

	// Open listener.
	ln, err := net.Listen("tcp", s.addr)
	if err != nil {
		return err
	}

	s.ln = ln

	go func() {
		err := s.server.Serve(s.ln)
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			s.logger.Error("httpd serve error", zap.Error(err))
		}
	}()
	s.logger.Info("httpd service started", zap.String("listen", s.addr))
	return nil
}

// Close closes the service.
func (s *Service) Close(ctx context.Context) error {
	return s.server.Shutdown(ctx)
}

func (s *Service) initHandler() {
	v1 := s.router.Group("/api/v1/")
	// remote write API
	v1.POST("prom/write", s.ServePromWrite)

	// query proxy API
	v1.GET("query", s.ProxyQuery)
	v1.POST("query", s.ProxyQuery)

	v1.GET("query_range", s.ProxyQuery)
	v1.POST("query_range", s.ProxyQuery)

	v1.GET("label/:name/values", s.ProxyQuery)

	v1.GET("series", s.ProxyQuery)
	v1.POST("series", s.ProxyQuery)

	v1.GET("labels", s.ProxyQuery)
	v1.POST("labels", s.ProxyQuery)

	// Handlers for pushing and deleting metrics.
	pushAPIPath := "/metrics"
	for _, suffix := range []string{"", Base64Suffix} {
		jobBase64Encoded := suffix == Base64Suffix
		s.router.PUT(pushAPIPath+"/job"+suffix+"/:job/*labels", s.Push(jobBase64Encoded))
		s.router.POST(pushAPIPath+"/job"+suffix+"/:job/*labels", s.Push(jobBase64Encoded))
		s.router.DELETE(pushAPIPath+"/job"+suffix+"/:job/*labels", s.Delete(jobBase64Encoded))
		s.router.PUT(pushAPIPath+"/job"+suffix+"/:job", s.Push(jobBase64Encoded))
		s.router.POST(pushAPIPath+"/job"+suffix+"/:job", s.Push(jobBase64Encoded))
		s.router.DELETE(pushAPIPath+"/job"+suffix+"/:job", s.Delete(jobBase64Encoded))
	}
}

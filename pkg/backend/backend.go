package backend

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/promcluster/proxy/pkg/backend/consistent"
	"github.com/promcluster/proxy/pkg/dns"

	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
)

var namespace = "promclusterproxy"
var subsystem = "backend"

var (
	SDDNSFailed = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "sd_dns_failed",
			Help:      "The failed number of dns resolved.",
		},
		[]string{"name"},
	)
	SDDNSSuccess = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "sd_dns_success",
			Help:      "The success number of dns resolved.",
		},
		[]string{"name"},
	)
	EndpointNum = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "endpoint_num",
			Help:      "The endpoints number of backend.",
		},
	)
)

// Backend represents backend server's endpoint.
type Backend interface {
	Endpoints(hash string, rep int) ([]Endpoint, error)
}

// PromServer implments Backend interface.
type PromServer struct {
	name        string
	c           consistent.Consistent
	provider    *dns.Provider
	interval    time.Duration
	concurrency int

	endpoints  map[string]Endpoint
	mu         sync.RWMutex
	registerer prometheus.Registerer
	logger     *zap.Logger
}

// NewPromServer creates a new PromServer.
func NewPromServer(
	ctx context.Context,
	reg prometheus.Registerer,
	name string,
	concurrency int,
	interval time.Duration,
	logger *zap.Logger) *PromServer {
	reg.MustRegister(SDDNSFailed, SDDNSSuccess, EndpointNum, EndpointSendFailed, EndpointSendSuccess)
	p := &PromServer{
		c:           consistent.NewCrc32(),
		name:        name,
		concurrency: concurrency,
		provider:    dns.NewProvider("golang", logger),
		interval:    interval,
		endpoints:   make(map[string]Endpoint),
		registerer:  reg,
		logger:      logger.With(zap.String("service", "backend")),
	}

	go p.refreshDNS(ctx)
	return p
}

// Endpoints implments backend interface.
func (p *PromServer) Endpoints(hash string, rep int) ([]Endpoint, error) {
	// if rep > members num, rep = members num.
	addrs, err := p.c.GetN(hash, rep)
	if err != nil {
		return nil, err
	}
	// get endpoint from promserver struct
	var res []Endpoint
	p.mu.RLock()
	defer p.mu.RUnlock()
	for _, addr := range addrs {
		if e, ok := p.endpoints[addr]; ok {
			res = append(res, e)
		} else {
			return nil, fmt.Errorf("not found endpoint for addr: %s", addr)
		}
	}
	return res, nil
}

func (p *PromServer) refreshDNS(ctx context.Context) {
	if err := p.resolve(ctx); err != nil {
		p.logger.Error("init DNS resolve", zap.Error(err))
	}

	t := time.NewTicker(p.interval)
	for {
		select {
		case <-t.C:
			err := p.resolve(ctx)
			if err != nil {
				p.logger.Error("DNS resolve", zap.Error(err))
				continue
			}
		case <-ctx.Done():
			p.closeEndpoints()
			return
		}
	}
}

func (p *PromServer) closeEndpoints() {
	p.mu.Lock()
	defer p.mu.Unlock()
	for k, e := range p.endpoints {
		e.Stop()
		delete(p.endpoints, k)
	}
}

func (p *PromServer) resolve(ctx context.Context) error {
	res, err := p.provider.Resolve(ctx, p.name)
	if err != nil {
		SDDNSFailed.WithLabelValues(p.name).Inc()
		return err
	}
	p.mu.Lock()
	seen := make(map[string]struct{}, len(res))
	for _, addr := range res {
		seen[addr] = struct{}{}
		if _, ok := p.endpoints[addr]; !ok {
			e := NewHTTPEndpoint(addr, p.concurrency, p.logger)
			go e.Start()
			p.endpoints[addr] = e
		}
	}

	// remove not exists any more
	for k, e := range p.endpoints {
		if _, ok := seen[k]; !ok {
			e.Stop()
			delete(p.endpoints, k)
		}
	}
	p.c.Set(res)
	EndpointNum.Set(float64(len(p.endpoints)))
	p.mu.Unlock()

	SDDNSSuccess.WithLabelValues(p.name).Inc()
	p.logger.Info("DNS resolve", zap.String("recoreds:", strings.Join(res, ",")))
	return nil
}

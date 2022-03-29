package dns

import (
	"context"
	"fmt"
	"net"
	"strings"
	"sync"

	"go.uber.org/zap"
)

// Provider is a stateful cache for asynchronous DNS resolutions.
// It provides a way to resolve addresses and obtain them.
type Provider struct {
	sync.RWMutex
	resolver Resolver
	// A map from domain name to a slice of resolved targets.
	resolved map[string][]string
	logger   *zap.Logger
}

type ResolverType string

const (
	GolangResolverType ResolverType = "golang"
)

func (t ResolverType) toResolver(logger *zap.Logger) ipLookupResolver {
	var r ipLookupResolver
	switch t {
	case GolangResolverType:
		r = net.DefaultResolver
	default:
		logger.Error("no such resolver type, defaulting to golang", zap.String("type", string(t)))
		r = net.DefaultResolver
	}
	return r
}

// NewProvider returns a new empty provider with a given resolver type.
// If empty resolver type is net.DefaultResolver.w
func NewProvider(resolverType ResolverType, logger *zap.Logger) *Provider {
	p := &Provider{
		resolver: NewResolver(resolverType.toResolver(logger)),
		resolved: make(map[string][]string),
		logger:   logger.With(zap.String("service", "DNS")),
	}
	return p
}

// GetQTypeName splits the provided addr into two parts: the QType (if any)
// and the name.
func GetQTypeName(addr string) (qtype string, name string) {
	qtypeAndName := strings.SplitN(addr, "+", 2)
	if len(qtypeAndName) != 2 {
		return "", addr
	}
	return qtypeAndName[0], qtypeAndName[1]
}

// Resolve stores a provided addresse or their DNS records if requested.
// Addresses prefixed with `dns+` or `dnssrv+` will be resolved through respective DNS lookup (A/AAAA or SRV).
// defaultPort is used for non-SRV records when a port is not supplied.
func (p *Provider) Resolve(ctx context.Context, addr string) ([]string, error) {
	var resolved []string
	qtype, name := GetQTypeName(addr)
	if qtype == "" {
		return nil, fmt.Errorf("unknown DNS query type")
	}

	resolved, err := p.resolver.Resolve(ctx, name, QType(qtype))
	if err != nil {
		p.logger.Error("resolve dns, try use cache", zap.Error(err))
		// Use cached values.
		var ok bool
		p.RLock()
		resolved, ok = p.resolved[addr]
		p.RUnlock()
		if !ok {
			return nil, err
		}
		return resolved, nil
	}

	p.Lock()
	p.resolved[addr] = resolved
	p.Unlock()

	return resolved, nil
}

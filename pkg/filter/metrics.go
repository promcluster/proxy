package filter

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/cespare/xxhash/v2"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/model"
	bloom "github.com/steakknife/bloomfilter"
)

var namespace = "promclusterproxy"
var subsystem = "filter"

const probCollide = 0.000001

var (
	SeriesLimit = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "series_limit",
			Help:      "The series limit.",
		},
		[]string{},
	)
	SeriesTotal = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "series_total",
			Help:      "The total number of this instance series met.",
		},
		[]string{},
	)
)

type MetricsFilter struct { //nolint: maligned
	disabled       bool
	bloom          *bloom.Filter
	mu             sync.RWMutex
	maxSeriesCount uint64
	flushInterval  time.Duration
	seriesCount    uint64
	registerer     prometheus.Registerer

	done chan struct{}
}

// NewMetricsFilter creates a new metrics filter
func NewMetricsFilter(reg prometheus.Registerer,
	maxSeries uint64,
	flushInterval time.Duration) (*MetricsFilter, error) {
	if maxSeries == 0 {
		return &MetricsFilter{disabled: true}, nil
	}
	reg.MustRegister(SeriesTotal)
	reg.MustRegister(SeriesLimit)
	SeriesLimit.WithLabelValues().Set(float64(maxSeries))

	bf, err := bloom.NewOptimal(maxSeries, probCollide)
	if err != nil {
		return nil, err
	}
	return &MetricsFilter{
		bloom:          bf,
		maxSeriesCount: maxSeries,
		flushInterval:  flushInterval,
		registerer:     reg,
		done:           make(chan struct{}),
	}, nil
}

// once for flush series count goroutine.
var once sync.Once

// Filt implements Filter
func (m *MetricsFilter) Filt(labels *model.LabelSet) error {
	if m.disabled {
		return nil
	}
	hash := xxhash.New()
	if _, err := hash.WriteString(LabelsString(labels)); err != nil {
		return nil
	}
	if m.bloom.Contains(hash) {
		return nil
	}
	c := atomic.LoadUint64(&m.seriesCount)
	if c > m.maxSeriesCount && m.maxSeriesCount != 0 {
		return fmt.Errorf("the maximum series count limit exceeded: %d", m.maxSeriesCount)
	}

	once.Do(func() { m.Purge() })

	m.mu.RLock()
	m.bloom.Add(hash)
	SeriesTotal.WithLabelValues().Inc()
	atomic.AddUint64(&m.seriesCount, 1)
	m.mu.RUnlock()
	return nil
}

// Purge resets series counter.
func (m *MetricsFilter) Purge() {
	ticker := time.NewTicker(m.flushInterval)
	var err error
	go func() {
		for {
			select {
			case <-m.done:
				ticker.Stop()
				return
			case <-ticker.C:
				m.mu.Lock()
				SeriesTotal.WithLabelValues().Set(0)
				m.bloom, err = bloom.NewOptimal(m.maxSeriesCount, probCollide)
				if err != nil {
					panic(err)
				}
				atomic.StoreUint64(&m.seriesCount, 0)
				m.mu.Unlock()
			}
		}
	}()
}

// LabelsString returns a labelset string efficiently.
func LabelsString(labels *model.LabelSet) string {
	var b strings.Builder
	lstrs := make([]string, 0, len(*labels))
	for l, v := range *labels {
		b.WriteString(string(l))
		b.WriteString("=\"")
		b.WriteString(string(v))
		b.WriteString("\"")
		lstrs = append(lstrs, b.String())
		b.Reset()
	}

	sort.Strings(lstrs)
	return "{" + strings.Join(lstrs, ", ") + "}"
}

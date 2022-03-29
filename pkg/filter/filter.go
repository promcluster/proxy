package filter

import (
	"github.com/prometheus/common/model"
)

// Filter implements prometheus metric labels filter.
// Cosumers will ignore this message with the lables if
// error returned.
type Filter interface {
	Filt(labels *model.LabelSet) error
}

type emptyFilter struct{}

// NewEmptyFilter creates an empty Filter,
// just for testing.
func NewEmptyFilter() *emptyFilter { //nolint: golint
	return &emptyFilter{}
}

// Filt implements Filter.
func (e *emptyFilter) Filt(labels *model.LabelSet) error {
	return nil
}

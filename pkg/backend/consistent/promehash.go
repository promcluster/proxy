package consistent

import (
	"encoding/json"

	"github.com/prometheus/common/model"
)

// PromShard represents prometheus hash
type PromShard struct {
	servers []string
}

// NewPromShard returns a new prometheus hash
func NewPromShard() *PromShard {
	return &PromShard{}
}

// GetN implements consistent interface
func (p *PromShard) GetN(key string, rep int) ([]string, error) {
	var lbset model.LabelSet
	err := json.Unmarshal([]byte(key), &lbset)
	if err != nil {
		return nil, err
	}
	fp := lbset.Fingerprint()
	shard := uint64(fp) % uint64(len(p.servers))
	return []string{p.servers[shard]}, nil
}

// Set implements consistent interface
func (p *PromShard) Set(s []string) {
	p.servers = s
}

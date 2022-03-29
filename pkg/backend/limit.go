package backend

import (
	"sync"
)

type job chan struct{}

// Limit represents a Limiter.
type Limit struct {
	sync.WaitGroup
	limit chan struct{}
	Err   chan error
}

// NewLimit creates a new limit.
func NewLimit(limit int) Limit {
	return Limit{limit: make(job, limit), Err: make(chan error, limit)}
}

// Take takes a process
func (l *Limit) Take() {
	l.Add(1)
	l.limit <- struct{}{}
}

// Release releases a process
func (l *Limit) Release() {
	l.Done()
	<-l.limit
}

// Error returns an error
func (l *Limit) Error(err error) {
	l.Err <- err
}

// Close closes the limiter
func (l *Limit) Close() {
	close(l.limit)
	close(l.Err)
}

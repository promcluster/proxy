package queue

import (
	"errors"

	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
)

var ErrQueueIsFull = errors.New("queue is full, discard message")
var DefaultQueueSize = 10000

// Queue is a queue for storing messages.
// Please keep these interfaces goroutine safe.
type Queue interface {
	// Push pushes message into the queue.
	Push([]byte) error

	// Pop pops a message from the queue.
	// if no message, block it until new message
	// push into the queue.
	Pop() ([]byte, error)
}

// ChanQueue implements a simple queue.
type ChanQueue struct {
	C chan []byte
}

// NewChanQueue creates a chan queue.
func NewChanQueue(reg prometheus.Registerer, logger *zap.Logger) *ChanQueue {
	return &ChanQueue{
		C: make(chan []byte, DefaultQueueSize),
	}
}

// Push pushes message into the queue.
func (c *ChanQueue) Push(msg []byte) error {
	select {
	case c.C <- msg:
	default:
		return ErrQueueIsFull
	}
	return nil
}

// Pop pops a message from the queue.
func (c *ChanQueue) Pop() ([]byte, error) {
	msg := <-c.C
	return msg, nil
}

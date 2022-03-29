package consumer

import (
	"log"
)

// Consumer is the message processing interface
//
// Implement this interface for handlers that return whether or not message
// processing completed successfully.
//
// When the return value is nil error will automatically handle FINishing.
// When the returned bool value is true will automatically handle REQueing.
type Consumer interface {
	HandleMessage([]byte) (bool, error)
}

// LogConsumer for testing
type LogConsumer struct{}

// NewLogConsumer returns a new log print consumer.
// It just for testing.
func NewLogConsumer() *LogConsumer {
	return &LogConsumer{}
}

// HandleMessage implements Consumer.
func (l *LogConsumer) HandleMessage(msg []byte) (bool, error) {
	log.Printf("consum message size: %d", len(msg))
	return false, nil
}

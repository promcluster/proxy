package consumer

import (
	"testing"
)

func TestLogConsumer(t *testing.T) {
	l := NewLogConsumer()
	_, err := l.HandleMessage([]byte("test"))
	if err != nil {
		t.Fatal(err)
	}
}

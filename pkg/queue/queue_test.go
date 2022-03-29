package queue

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
)

func TestChanQueue(t *testing.T) {
	q := NewChanQueue(prometheus.DefaultRegisterer, zap.NewExample())
	msg := []byte("test")
	if err := q.Push(msg); err != nil {
		t.Fatal(err)
	}

	got, err := q.Pop()
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(msg) {
		t.Fatal("got: " + string(got) + ", but want: " + string(msg))
	}
}

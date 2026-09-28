package event

import (
	"testing"
)

type TestEvent struct {
	Value int
}

func TestBus(t *testing.T) {
	b := NewBus()

	got := &TestEvent{
		Value: 43,
	}
	Subscribe(b, func(e *TestEvent) {
		got.Value = e.Value
	})

	if got.Value != 43 {
		t.Errorf("want 43 got %d", got)
	}

	Publish(b, &TestEvent{Value: 64})

	// got should now have been changed
	if got.Value != 64 {
		t.Errorf("want 64 got %d", got)
	}
}

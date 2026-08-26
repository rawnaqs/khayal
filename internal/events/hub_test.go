package events

import (
	"sync"
	"testing"

	"github.com/rawnaqs/khayal/internal/queue"
)

func TestHubFanOut(t *testing.T) {
	h := NewHub()
	a := h.Subscribe()
	b := h.Subscribe()

	evt := Event{Event: "job_updated", Job: &queue.Job{ID: "j1", Status: "done"}}
	h.Publish(evt)

	for name, ch := range map[string]chan Event{"a": a, "b": b} {
		got := <-ch
		if got.Event != "job_updated" || got.Job.ID != "j1" {
			t.Errorf("%s got %+v", name, got)
		}
	}

	h.Unsubscribe(a)
	if _, ok := <-a; ok {
		t.Error("unsubscribed channel should be closed")
	}
	// b still receives
	h.Publish(evt)
	if got := <-b; got.Job.ID != "j1" {
		t.Errorf("b lost after unsubscribe of a: %+v", got)
	}
}

func TestHubPublishAfterUnsubscribeDoesNotPanic(t *testing.T) {
	h := NewHub()
	ch := h.Subscribe()
	h.Unsubscribe(ch)
	h.Publish(Event{Event: "job_updated", Job: nil}) // must not panic on closed channel
	_ = ch
}

func TestHubSlowConsumerIsDropped(t *testing.T) {
	h := NewHub()
	_ = h.Subscribe() // buffer 16, never drained

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		job := &queue.Job{ID: "x"}
		for i := 0; i < 1000; i++ {
			h.Publish(Event{Event: "job_updated", Job: job})
		}
	}()
	wg.Wait() // would deadlock if Publish blocked on the full channel
}

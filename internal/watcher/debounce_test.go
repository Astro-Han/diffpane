package watcher

import (
	"testing"
	"time"
)

// TestDebouncerSingleEvent verifies one trigger eventually fires one callback.
func TestDebouncerSingleEvent(t *testing.T) {
	fired := make(chan struct{}, 1)
	debouncer := NewDebouncer(50*time.Millisecond, 200*time.Millisecond, func() {
		fired <- struct{}{}
	})
	defer debouncer.Stop()

	debouncer.Trigger()

	select {
	case <-fired:
	case <-time.After(300 * time.Millisecond):
		t.Fatal("callback did not fire")
	}
}

// TestDebouncerRapidEvents verifies rapid triggers collapse into one callback.
func TestDebouncerRapidEvents(t *testing.T) {
	count := 0
	done := make(chan struct{}, 10)
	debouncer := NewDebouncer(50*time.Millisecond, 200*time.Millisecond, func() {
		count++
		done <- struct{}{}
	})
	defer debouncer.Stop()

	for range 5 {
		debouncer.Trigger()
		time.Sleep(10 * time.Millisecond)
	}

	select {
	case <-done:
	case <-time.After(300 * time.Millisecond):
		t.Fatal("callback did not fire")
	}
	time.Sleep(100 * time.Millisecond)
	if count != 1 {
		t.Fatalf("callback count = %d, want 1", count)
	}
}

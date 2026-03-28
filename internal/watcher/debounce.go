package watcher

import (
	"sync"
	"time"
)

// Debouncer combines debounce and throttle behavior for watcher callbacks.
type Debouncer struct {
	delay    time.Duration
	interval time.Duration
	callback func()

	mu       sync.Mutex
	timer    *time.Timer
	lastFire time.Time
	stopped  bool
}

// NewDebouncer constructs a debouncer with one shared callback.
func NewDebouncer(delay, interval time.Duration, callback func()) *Debouncer {
	return &Debouncer{
		delay:    delay,
		interval: interval,
		callback: callback,
	}
}

// Trigger schedules the callback after the debounce window.
func (d *Debouncer) Trigger() {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.stopped {
		return
	}
	if d.timer != nil {
		d.timer.Stop()
	}
	d.timer = time.AfterFunc(d.delay, d.fire)
}

// fire enforces throttle timing before invoking the callback.
func (d *Debouncer) fire() {
	d.mu.Lock()
	if d.stopped {
		d.mu.Unlock()
		return
	}

	since := time.Since(d.lastFire)
	if since < d.interval {
		wait := d.interval - since
		d.timer = time.AfterFunc(wait, d.fire)
		d.mu.Unlock()
		return
	}

	d.lastFire = time.Now()
	d.mu.Unlock()
	d.callback()
}

// Stop cancels future callbacks and releases any pending timer.
func (d *Debouncer) Stop() {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.stopped = true
	if d.timer != nil {
		d.timer.Stop()
	}
}

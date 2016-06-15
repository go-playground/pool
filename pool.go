package pool

import (
	"fmt"
	"math"
	"runtime"
	"sync"
	"sync/atomic"
)

const (
	errCancelled = "ERROR: Work Unit Cancelled"
	errRecovery  = "ERROR: Work Unit failed due to a recoverable error: '%v'\n, Stack Trace:\n %s"
	errClosed    = "ERROR: Work Unit added/run after the pool had been closed or cancelled"
)

// ErrRecovery contains the error when a consumer goroutine needed to be recovers
type ErrRecovery struct {
	s string
}

// Error prints recovery error
func (e *ErrRecovery) Error() string {
	return e.s
}

// ErrPoolClosed is the error returned to all work units that may have been in or added to the pool after it's closing.
type ErrPoolClosed struct {
	s string
}

// Error prints Work Unit Close error
func (e *ErrPoolClosed) Error() string {
	return e.s
}

// ErrCancelled is the error returned to a Work Unit when it has been cancelled.
type ErrCancelled struct {
	s string
}

// Error prints Work Unit Cancellation error
func (e *ErrCancelled) Error() string {
	return e.s
}

// WorkUnit contains a single unit of works values
type WorkUnit struct {
	Value     interface{}
	Error     error
	Done      chan struct{}
	fn        WorkFunc
	cancelled atomic.Value
	running   atomic.Value
}

// Cancel cancels this specific unit of work.
func (wu *WorkUnit) Cancel() {
	wu.cancelWithError(&ErrCancelled{s: errCancelled})
}

func (wu *WorkUnit) cancelWithError(err error) {
	if wu.running.Load() == nil && wu.cancelled.Load() == nil {
		wu.cancelled.Store(struct{}{})
		wu.Error = err
		close(wu.Done)
	}
}

// WorkFunc is the function type needed by the pool
type WorkFunc func() (interface{}, error)

// Pool in the main pool instance.
type Pool struct {
	workers uint
	work    chan *WorkUnit
	cancel  chan struct{}
	closed  bool
	m       *sync.RWMutex
}

// New returns a new pool instance.
func New(workers uint) *Pool {

	if workers == 0 {
		panic("invalid workers '0'")
	}

	p := &Pool{
		workers: workers,
		m:       new(sync.RWMutex),
	}

	p.initialize()

	return p
}

func (p *Pool) initialize() {

	p.work = make(chan *WorkUnit, p.workers*2)
	p.cancel = make(chan struct{})
	p.closed = false

	// fire up workers here
	for i := 0; i < int(p.workers); i++ {
		p.newWorker(p.work, p.cancel)
	}
}

// passing work and cancel channels to newWorker() to avoid any potential race condition
// betweeen p.work read & write
func (p *Pool) newWorker(work chan *WorkUnit, cancel chan struct{}) {
	go func(p *Pool) {

		var wu *WorkUnit

		defer func(p *Pool) {
			if err := recover(); err != nil {

				trace := make([]byte, 1<<16)
				n := runtime.Stack(trace, true)

				s := fmt.Sprintf(errRecovery, err, string(trace[:int(math.Min(float64(n), float64(7000)))]))

				iwu := wu
				iwu.Error = &ErrRecovery{s: s}
				close(iwu.Done)

				// need to fire up new worker to replace this one as this one is exiting
				p.newWorker(p.work, p.cancel)
			}
		}(p)

		for {
			select {
			case wu = <-work:

				// possible for one more nilled out value to make it
				// through when channel closed, don't quite understad the why
				if wu == nil {
					continue
				}

				wu.running.Store(struct{}{})

				// support for individual WorkUnit cancellation
				// and batch job cancellation
				if wu.cancelled.Load() == nil {
					wu.Value, wu.Error = wu.fn()

					// who knows where the Done channel is being listened to on the other end
					// don't want this to block just because caller is waiting on another unit
					// of work to be done first so we use close
					close(wu.Done)
				}

			case <-cancel:
				return
			}
		}

	}(p)
}

// Queue queues the work to be run, and starts processing immediately
func (p *Pool) Queue(fn WorkFunc) *WorkUnit {

	w := &WorkUnit{
		Done: make(chan struct{}),
		fn:   fn,
	}

	go func() {
		p.m.RLock()
		if p.closed {
			w.Error = &ErrPoolClosed{s: errClosed}
			if w.cancelled.Load() == nil {
				close(w.Done)
			}
			p.m.RUnlock()
			return
		}

		p.work <- w

		p.m.RUnlock()
	}()

	return w
}

// Reset reinitializes a pool that has been closed/cancelled back to a working state.
// if the pool has not been closed/cancelled, nothing happens as the pool is still in
// a valid running state
func (p *Pool) Reset() {

	p.m.Lock()

	if !p.closed {
		p.m.Unlock()
		return
	}

	// cancelled the pool, not closed it, pool will be usable after calling initialize().
	p.initialize()
	p.m.Unlock()
}

func (p *Pool) closeWithError(err error) {

	p.m.Lock()

	if !p.closed {
		close(p.cancel)
		close(p.work)
		p.closed = true
	}

	for wu := range p.work {
		wu.cancelWithError(err)
	}

	p.m.Unlock()
}

// Cancel cleans up the pool workers and channels and cancels and pending
// work still yet to be processed.
// call Reset() to reinitialize the pool for use.
func (p *Pool) Cancel() {

	err := &ErrCancelled{s: errCancelled}
	p.closeWithError(err)
}

// Close cleans up the pool workers and channels and cancels any pending
// work still yet to be processed.
// call Reset() to reinitialize the pool for use.
func (p *Pool) Close() {

	err := &ErrPoolClosed{s: errClosed}
	p.closeWithError(err)
}

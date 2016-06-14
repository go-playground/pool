package pool

import (
	"fmt"
	"log"
	"runtime"
	"sync"
	"sync/atomic"
)

const (
	errWorkCancelled         = "Work Cancelled"
	errWorkCancelledRecovery = "Work Cancelled due to Work Unit Error, Stack Trace:\n %s"
	errWorkUnitClosed        = "Pool has been closed, work not run."
)

// WorkUnitCloseError is the error returned to all work units that may have been in or added to the pool after it's closing.
type WorkUnitCloseError struct {
	s string
}

// Error prints Work Unit Close error
func (wuc *WorkUnitCloseError) Error() string {
	return wuc.s
}

// WorkUnitCancelledErr is the error returned to all work units when a pool is cancelled.
type WorkUnitCancelledErr struct {
	s string
}

// Error prints Work Unit Cancelation error
func (wuc *WorkUnitCancelledErr) Error() string {
	return wuc.s
}

// WorkUnitCancelledRecoveryErr is the error returned to all work units when a pool is cancelled becaue of an recovery error.
type WorkUnitCancelledRecoveryErr struct {
	s string
}

// Error prints Work Unit Cancelation error that cause the worker to recover
func (wuc *WorkUnitCancelledRecoveryErr) Error() string {
	return wuc.s
}

// WorkUnit contains a single unit of works values
type WorkUnit struct {
	Value     interface{}
	Error     error
	Done      chan struct{}
	fn        WorkFunc
	cancelled atomic.Value
}

// Cancel cancels this specific unit of work.
func (wu *WorkUnit) Cancel() {
	wu.cancelWithError(&WorkUnitCancelledErr{s: errWorkCancelled})
}

func (wu *WorkUnit) cancelWithError(err error) {
	wu.cancelled.Store(struct{}{})
	wu.Error = err
	close(wu.Done)
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
		p.newWorker()
	}
}

func (p *Pool) newWorker() {
	go func(p *Pool) {

		var wu *WorkUnit

		defer func(p *Pool) {
			if err := recover(); err != nil {

				trace := make([]byte, 1<<16)
				n := runtime.Stack(trace, true)

				if n > 7000 {
					n = 7000
				}

				s := string(trace[:n])

				log.Println(s)

				iwu := wu
				iwu.cancelWithError(&WorkUnitCancelledRecoveryErr{s: fmt.Sprintf(errWorkCancelledRecovery, s)})

				// need to fire up new worker to replace this one as this one is exiting
				p.newWorker()
			}
		}(p)

		for {
			select {
			case wu = <-p.work:

				// possible for one more nilled out value to make it
				// through when channel closed, don't quite understad the why
				if wu == nil {
					continue
				}

				// support for individual WorkUnit cancellation
				// and batch job cancellation
				if wu.cancelled.Load() == nil {
					wu.Value, wu.Error = wu.fn()
				}

				// who knows where the Done channel is being listened to on the other end
				// don't want this to block just because caller is waiting on another unit
				// of work to be done first so we use close
				close(wu.Done)

			case <-p.cancel:
				return
			}
		}

	}(p)
}

// Queue queues the work to be run, and starts processing immediatly
func (p *Pool) Queue(fn WorkFunc) *WorkUnit {

	w := &WorkUnit{
		Done: make(chan struct{}),
		fn:   fn,
	}

	go func() {
		p.m.RLock()
		if p.closed {
			w.Error = &WorkUnitCloseError{s: errWorkUnitClosed}
			close(w.Done)
			p.m.RUnlock()
			return
		}
		p.m.RUnlock()

		p.work <- w
	}()

	return w
}

// Cancel cancels all jobs not already running.
// It can also be called from within a job through the Job object
func (p *Pool) Cancel() {

	p.m.Lock()

	err := &WorkUnitCancelledErr{s: errWorkCancelled}

	if !p.closed {
		close(p.cancel)
		close(p.work)
		p.closed = true
	}

	for wu := range p.work {
		wu.cancelWithError(err)
	}

	// cancelled the pool, not closed it, pool will be usable after calling initialize().
	p.initialize()
	p.m.Unlock()
}

// Close cleans up the pool workers and channels
func (p *Pool) Close() {

	p.m.Lock()

	if !p.closed {
		close(p.cancel)
		close(p.work)
		p.closed = true
	}

	err := &WorkUnitCloseError{s: errWorkUnitClosed}

	for wu := range p.work {
		wu.cancelWithError(err)
	}

	p.m.Unlock()
}

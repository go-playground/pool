package pool

import (
	"fmt"
	"log"
	"runtime"
	"sync"
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
	Value interface{}
	Error error
	Done  chan struct{}
}

// WorkFunc is the function type needed by the pool
type WorkFunc func() (interface{}, error)

type consumableWork struct {
	fn WorkFunc
	wu *WorkUnit
}

// Pool in the main pool instance.
type Pool struct {
	work   chan consumableWork
	cancel chan struct{}
	closed bool
	m      *sync.RWMutex
}

// New returns a new pool instance.
func New(workers uint) *Pool {

	if workers == 0 {
		panic("invalid workers '0'")
	}

	p := &Pool{
		work:   make(chan consumableWork, workers*2),
		cancel: make(chan struct{}),
		m:      new(sync.RWMutex),
	}

	// fire up workers here
	for i := 0; i < int(workers); i++ {
		go func(p *Pool) {

			var cw consumableWork

			defer func(p *Pool) {
				if err := recover(); err != nil {

					trace := make([]byte, 1<<16)
					n := runtime.Stack(trace, true)

					if n > 7000 {
						n = 7000
					}

					s := string(trace[:n])

					log.Println(s)
					p.cancelWithError(&WorkUnitCancelledRecoveryErr{s: fmt.Sprintf(errWorkCancelledRecovery, s)})
				}
			}(p)

			for {
				select {
				case cw = <-p.work:

					// possible for one more nilled out value to make it
					// through when channel closed, don't quite understad the why
					if cw.fn == nil {
						continue
					}

					cw.wu.Value, cw.wu.Error = cw.fn()

					// who knows where the Done channel is being listened to on the other end
					// don't want this to block just because caller is waiting on another unit
					// of work to be done first.
					go func(cw consumableWork) {
						cw.wu.Done <- struct{}{}
					}(cw)
				case <-p.cancel:
					return
				}
			}

		}(p)
	}

	return p
}

// Queue queues the work to be run, and starts processing immediatly
func (p *Pool) Queue(fn WorkFunc) *WorkUnit {

	w := &WorkUnit{
		Done: make(chan struct{}),
	}

	// p.m.RLock()
	// if p.closed == true {

	// 	fmt.Println("Closed")
	// 	go func() {
	// 		w.Error = &WorkUnitCloseError{s: errWorkUnitClosed}
	// 		w.Done <- struct{}{}
	// 	}()

	// 	p.m.RUnlock()
	// 	return w
	// }

	go func() {
		p.m.RLock()
		if p.closed {
			w.Error = &WorkUnitCloseError{s: errWorkUnitClosed}
			w.Done <- struct{}{}
			return
		}
		p.m.RUnlock()

		p.work <- consumableWork{fn: fn, wu: w}
	}()

	// p.m.RUnlock()

	return w
}

func (p *Pool) cancelWithError(err error) {

	close(p.cancel)

	p.m.Lock()
	p.closed = true
	p.m.Unlock()

	// fmt.Println(p.closed.Load() != nil)

	close(p.work)

	for cw := range p.work {
		go func(cw consumableWork) {
			cw.wu.Error = err
			cw.wu.Done <- struct{}{}
		}(cw)
	}
}

// Cancel cancels all jobs not already running.
// It can also be called from within a job through the Job object
func (p *Pool) Cancel() {
	p.cancelWithError(&WorkUnitCancelledErr{s: errWorkCancelled})
}

// Close cleans up the pool workers and channels
func (p *Pool) Close() {
	close(p.cancel)

	// p.closed.Store(struct{}{})
	p.m.Lock()
	p.closed = true
	p.m.Unlock()
	close(p.work)

	err := &WorkUnitCloseError{s: errWorkUnitClosed}

	for cw := range p.work {
		go func(cw consumableWork) {
			cw.wu.Error = err
			cw.wu.Done <- struct{}{}
		}(cw)
	}
}

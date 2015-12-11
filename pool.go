package pool

import (
	"fmt"
	"reflect"
	"runtime"
	"sync"
)

const (
	errRecoveryString = "recovering from panic: %+v\nStack Trace:\n %s"
)

// ConsumerHook type is a function that is called during the consumer startup
// and the return value is added to each Job just prior to firing off the job.
// This is good for say creating a database connection for every job to use but
// not having more than there are consumers.
type ConsumerHook func() interface{}

// ErrRecovery contains the error when a consumer goroutine needed to be recovers
type ErrRecovery struct {
	s string
}

// Error prints recovery error
func (e *ErrRecovery) Error() string {
	return e.s
}

// Pool Contains all information for the pool instance
type Pool struct {
	jobs         chan *Job
	results      chan interface{}
	cancel       chan struct{}
	wg           *sync.WaitGroup
	cancelled    bool
	cancelLock   sync.RWMutex
	consumerHook ConsumerHook
}

// JobFunc is the consumable function/job you wish to run
type JobFunc func(job *Job)

// Job contains all information to run a job
type Job struct {
	fn        JobFunc
	params    []interface{}
	hookParam interface{}
	pool      *Pool
}

// HookParam returns the value, if any, set by the ConsumerHook.
// Example a database connection.
func (j *Job) HookParam() interface{} {
	return j.hookParam
}

// Params returns an array of the params that were passed in during the Queueing of the funciton
func (j *Job) Params() []interface{} {
	return j.params
}

// Cancel is a way to let the pool know, from a job, that it should cancel the rest of the
// jobs to be run. The most likely scenario is because an error occured
func (j *Job) Cancel() {
	j.pool.Cancel()
}

// Return returns the jobs result
func (j *Job) Return(result interface{}) {
	j.pool.results <- result
}

// NewPool initializes and returns a new pool instance
func NewPool(consumers int, jobs int) *Pool {

	p := &Pool{
		wg:      new(sync.WaitGroup),
		jobs:    make(chan *Job, jobs),
		results: make(chan interface{}, jobs),
		cancel:  make(chan struct{}),
	}

	for i := 0; i < consumers; i++ {
		go func(p *Pool) {
			defer func(p *Pool) {
				if err := recover(); err != nil {
					trace := make([]byte, 1<<16)
					n := runtime.Stack(trace, true)
					rerr := &ErrRecovery{
						s: fmt.Sprintf(errRecoveryString, err, trace[:n]),
					}
					p.results <- rerr
					p.Cancel()
					p.wg.Done()
				}
			}(p)

			var consumerParm interface{}

			if p.consumerHook != nil {
				consumerParm = p.consumerHook()
			}

			for {
				select {
				case j := <-p.jobs:
					if reflect.ValueOf(j).IsNil() {
						return
					}

					j.hookParam = consumerParm
					j.fn(j)
					p.wg.Done()
				case <-p.cancel:
					return
				}
			}
		}(p)
	}
	return p
}

// AddConsumerHook registers a Consumer Hook function to be called by every consumer
// and setting the return value on every job prior to running. Use case is for
// reusing database connections.
func (p *Pool) AddConsumerHook(fn ConsumerHook) {
	p.consumerHook = fn
}

func (p *Pool) cancelJobs() {
	for range p.jobs {
		p.wg.Done()
	}
}

// Queue adds a job to be processed and the params to be passed to it.
func (p *Pool) Queue(fn JobFunc, params ...interface{}) {

	p.cancelLock.Lock()
	defer p.cancelLock.Unlock()

	if p.cancelled {
		return
	}

	job := &Job{
		fn:     fn,
		params: params,
		pool:   p,
	}

	p.wg.Add(1)
	p.jobs <- job
}

// Cancel cancels all jobs not already running.
// It can also be called from within a job through the Job object
func (p *Pool) Cancel() {
	close(p.cancel)
	p.cancelLock.Lock()
	p.cancelled = true
	p.cancelLock.Unlock()
	p.cancelJobs()
}

// Results returns the processed job results
func (p *Pool) Results() <-chan interface{} {

	close(p.jobs)

	go func() {
		p.wg.Wait()
		close(p.results)
	}()

	return p.results
}

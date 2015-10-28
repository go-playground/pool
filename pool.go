package pool

import (
	"reflect"
	"sync"
)

// Pool Contains channels to instantiate pool
type Pool struct {
	jobs       chan *Job
	results    chan interface{}
	cancel     chan struct{}
	wg         *sync.WaitGroup
	cancelled  bool
	cancelLock sync.RWMutex
}

// JobFunc ...
type JobFunc func(job *Job)

// ... make this into sync.Pool
type Job struct {
	fn     JobFunc
	params []interface{}
	pool   *Pool
}

func (j *Job) Params() []interface{} {
	return j.params
}

func (j *Job) Cancel() {
	j.pool.Cancel()
}

func (j *Job) Return(result interface{}) {
	j.pool.results <- result
}

// NewPool initializes pool
func NewPool(consumers int, jobs int) *Pool {

	// make this a sync pool as well
	p := &Pool{
		wg:      new(sync.WaitGroup),
		jobs:    make(chan *Job, jobs),
		results: make(chan interface{}, jobs),
		cancel:  make(chan struct{}),
	}

	for i := 0; i < consumers; i++ {
		go func(p *Pool) {
			// defer fmt.Println("GOROUTINE DIE")
			for {
				select {
				case j := <-p.jobs:
					if reflect.ValueOf(j).IsNil() {
						return
					}
					defer p.wg.Done()
					// log.Println("Running")
					j.fn(j)
				case <-p.cancel:
					// fmt.Println("Cancelling")
					return
				}
			}
		}(p)
	}
	return p
}

func (p *Pool) cancelJobs() {
	for range p.jobs {
		p.wg.Done()
	}
}

// Queue ...
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

// Cancel cancels all jobs not already running
func (p *Pool) Cancel() {
	close(p.cancel)
	p.cancelLock.Lock()
	p.cancelled = true
	p.cancelLock.Unlock()
	p.cancelJobs()
}

// Results ...
func (p *Pool) Results() <-chan interface{} {

	close(p.jobs)

	go func() {
		p.wg.Wait()
		close(p.results)
	}()

	// log.Println("Returning Results Channel")

	return p.results
}

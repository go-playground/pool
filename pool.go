package pool

import (
	"reflect"
	"sync"
)

// Pool Contains channels to instantiate pool
type Pool struct {
	jobs          chan *Job
	results       chan interface{}
	cancel        chan struct{}
	wg            *sync.WaitGroup
	jobsClosed    bool
	resultsClosed bool
	jobsLock      sync.RWMutex
	resultsLock   sync.RWMutex
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

	j.pool.resultsLock.Lock()
	defer j.pool.resultsLock.Unlock()

	if j.pool.resultsClosed {
		return
	}

	j.pool.results <- result
}

// NewPool initializes pool
func NewPool(consumers int, jobs int) *Pool {

	// make this a sync pool as well
	p := &Pool{
		wg:            new(sync.WaitGroup),
		jobs:          make(chan *Job, jobs),
		results:       make(chan interface{}, jobs),
		cancel:        make(chan struct{}),
		jobsClosed:    false,
		resultsClosed: false,
	}

	for i := 0; i < consumers; i++ {
		go func(p *Pool) {
			for {
				select {
				case j := <-p.jobs:
					if reflect.ValueOf(j).IsNil() {
						return
					}
					defer p.wg.Done()
					// log.Println("Running")
					j.fn(j)
					// j.fn(p.results, p.cancel, c.param...)
				case <-p.cancel:
					// fmt.Println("Cancelling")
					p.cancelJobs()
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

	p.closeJobsChannel()
	p.closeResultsChannel()
}

func (p *Pool) closeJobsChannel() {
	p.jobsLock.Lock()
	defer p.jobsLock.Unlock()
	if !p.jobsClosed {
		p.jobsClosed = true
		close(p.jobs)
	}
}

func (p *Pool) closeResultsChannel() {
	p.resultsLock.Lock()
	defer p.resultsLock.Unlock()
	if !p.resultsClosed {
		p.resultsClosed = true
		close(p.results)
	}
}

// Queue ...
func (p *Pool) Queue(fn JobFunc, params ...interface{}) {

	job := &Job{
		fn:     fn,
		params: params,
		pool:   p,
	}

	p.jobsLock.Lock()
	defer p.jobsLock.Unlock()

	if p.jobsClosed {
		return
	}

	p.wg.Add(1)
	p.jobs <- job
}

// Cancel cancels all jobs not already running
func (p *Pool) Cancel() {
	p.cancel <- struct{}{}
	p.cancelJobs()
}

// Results ...
func (p *Pool) Results() <-chan interface{} {

	p.closeJobsChannel()

	go func() {
		// log.Println("Waiting for completion")
		p.wg.Wait()
		// fmt.Println("Closing WAIT FUNCTION!")
		p.closeResultsChannel()
	}()

	// log.Println("Returning Results Channel")

	return p.results
}

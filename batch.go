package pool

import "sync"

// Batch contains all information for a batch run of WorkUnits
type Batch struct {
	pool    *Pool
	m       *sync.Mutex
	units   []*WorkUnit
	results chan *WorkUnit
	done    chan struct{}
	closed  bool
	wg      *sync.WaitGroup
}

// Batch creates a new Batch object for queueing Work Units separate from any others
// that may be running on the pool. Grouping these Work Units together allows for individual
// Cancellation of the Batch Work Units without affecting anything else running on the pool
// as well as outputting the results on a channel as they complete.
// NOTE: Batch is not reusable, once QueueComplete() has been called it's lifetime has been sealed
// to completing the Queued items.
func (p *Pool) Batch() *Batch {
	return &Batch{
		pool:    p,
		m:       new(sync.Mutex),
		units:   make([]*WorkUnit, 0, 4), // capacity it to 4 so it doesn't grow and allocate too many times.
		results: make(chan *WorkUnit),
		done:    make(chan struct{}),
		wg:      new(sync.WaitGroup),
	}
}

// Queue queues the work to be run in the pool and starts processing immediatly
// and also retains a reference for Cancellation and outputting to results.
// WARNING be sure to call QueueComplete() once all work has been Queued.
func (b *Batch) Queue(fn WorkFunc) {

	b.m.Lock()

	if b.closed {
		return
	}

	wu := b.pool.Queue(fn)

	b.units = append(b.units, wu) // keeping a reference for cancellation purposes
	b.wg.Add(1)
	b.m.Unlock()

	go func(b *Batch, wu *WorkUnit) {
		<-wu.Done
		b.results <- wu
		b.wg.Done()
	}(b, wu)
}

// QueueComplete lets the batch know that there will be no more Work Units Queued
// so that it may close the results channels once all work is completed.
// WARNING: if this function is not called the results channel will never exhaust,
// but block forever listening for more results.
func (b *Batch) QueueComplete() {
	b.m.Lock()
	b.closed = true
	close(b.done)
	b.m.Unlock()
}

// Cancel cancells the Work Units belonging to this Batch
func (b *Batch) Cancel() {

	b.QueueComplete() // no more to be added

	b.m.Lock()

	// go in reverse order to try and cancel as amany as possbile
	// one at end are less likely to have run than those at the beginning
	for i := len(b.units) - 1; i >= 0; i-- {
		b.units[i].Cancel()
	}

	b.m.Unlock()
}

// Results returns a Work Unit result channel that will output all
// completed units of work.
func (b *Batch) Results() <-chan *WorkUnit {

	go func(b *Batch) {
		<-b.done
		b.m.Lock()
		b.wg.Wait()
		b.m.Unlock()
		close(b.results)
	}(b)

	return b.results
}

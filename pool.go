package pool

// Pool contains all information for a pool instance.
type Pool interface {
	Queue(fn WorkFunc) WorkUnit
	Reset()
	Cancel()
	Close()
	Batch() Batch
}

// WorkFunc is the function type needed by the pool
type WorkFunc func(wu WorkUnit) (interface{}, error)

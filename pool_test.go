package pool

import (
	"os"
	"testing"
	"time"

	. "gopkg.in/go-playground/assert.v1"
)

// NOTES:
// - Run "go test" to run tests
// - Run "gocov test | gocov report" to report on test converage by file
// - Run "gocov test | gocov annotate -" to report on all code and functions, those ,marked with "MISS" were never called
//
// or
//
// -- may be a good idea to change to output path to somewherelike /tmp
// go test -coverprofile cover.out && go tool cover -html=cover.out -o cover.html
//

// global pool for testing long running pool
var gpool *Pool

func TestMain(m *testing.M) {

	// setup
	gpool = New(4)
	defer gpool.Close()

	os.Exit(m.Run())

	// teardown
}

func TestPool(t *testing.T) {

	var res []*WorkUnit

	pool := New(4)
	defer pool.Close()

	newFunc := func(d time.Duration) WorkFunc {
		return func() (interface{}, error) {
			time.Sleep(d)
			return nil, nil
		}
	}

	for i := 0; i < 4; i++ {
		wu := pool.Queue(newFunc(time.Second * 1))
		res = append(res, wu)
	}

	var count int

	for _, wu := range res {
		<-wu.Done
		Equal(t, wu.Error, nil)
		Equal(t, wu.Value, nil)
		count++
	}

	Equal(t, count, 4)

	pool.Close() // testing no error occurs as Close will be called twice once defer pool.Close() fires
}

func TestCancel(t *testing.T) {

	var res []*WorkUnit

	pool := gpool
	defer pool.Close()

	newFunc := func(d time.Duration) WorkFunc {
		return func() (interface{}, error) {
			time.Sleep(d)
			return 1, nil
		}
	}

	for i := 0; i < 125; i++ {
		wu := pool.Queue(newFunc(time.Second * 1))
		res = append(res, wu)
	}

	pool.Cancel()

	var count int

	for _, wu := range res {
		<-wu.Done

		if wu.Error != nil {
			_, ok := wu.Error.(*ErrCancelled)
			if !ok {
				_, ok = wu.Error.(*ErrPoolClosed)
				if ok {
					Equal(t, wu.Error.Error(), "ERROR: Work Unit added/run after the pool had been closed or cancelled")
				}
			} else {
				Equal(t, wu.Error.Error(), "ERROR: Work Unit Cancelled")
			}
			Equal(t, ok, true)
			continue
		}

		count += wu.Value.(int)
	}

	NotEqual(t, count, 40)

	// reset and test again
	pool.Reset()

	wrk := pool.Queue(newFunc(time.Millisecond * 300))
	<-wrk.Done

	_, ok := wrk.Value.(int)
	Equal(t, ok, true)

	wrk = pool.Queue(newFunc(time.Millisecond * 300))
	time.Sleep(time.Second * 1)
	wrk.Cancel()
	<-wrk.Done // proving we don't get stuck here after cancel
	Equal(t, wrk.Error, nil)

	pool.Reset() // testing that we can do this and nothing bad will happen as it checks if pool closed
}

func TestPanicRecovery(t *testing.T) {

	pool := New(2)
	defer pool.Close()

	newFunc := func(d time.Duration, i int) WorkFunc {
		return func() (interface{}, error) {
			if i == 1 {
				panic("OMG OMG OMG! something bad happened!")
			}
			time.Sleep(d)
			return 1, nil
		}
	}

	var wrk *WorkUnit
	for i := 0; i < 4; i++ {
		time.Sleep(time.Second * 1)
		if i == 1 {
			wrk = pool.Queue(newFunc(time.Second*1, i))
			continue
		}
		pool.Queue(newFunc(time.Second*1, i))
	}
	<-wrk.Done

	NotEqual(t, wrk.Error, nil)
	Equal(t, wrk.Error.Error()[0:90], "ERROR: Work Unit failed due to a recoverable error: 'OMG OMG OMG! something bad happened!'")

}

func TestBadWorkerCount(t *testing.T) {
	PanicMatches(t, func() { New(0) }, "invalid workers '0'")
}

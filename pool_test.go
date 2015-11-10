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

func TestMain(m *testing.M) {

	// setup

	os.Exit(m.Run())

	// teardown
}

func TestPool(t *testing.T) {

	pool := NewPool(4, 4)

	fn := func(job *Job) {

		i := job.Params()[0].(int)
		time.Sleep(time.Second * 1)
		job.Return(i)
	}

	for i := 0; i < 4; i++ {
		pool.Queue(fn, i)
	}

	var count int

	for range pool.Results() {
		count++
	}

	Equal(t, count, 4)
}

func TestCancel(t *testing.T) {

	pool := NewPool(2, 4)

	fn := func(job *Job) {

		i := job.Params()[0].(int)
		if i == 1 {
			job.Cancel()
			return
		}
		time.Sleep(time.Second * 1)
		job.Return(i)
	}

	for i := 0; i < 4; i++ {
		pool.Queue(fn, i)
	}

	var count int

	for range pool.Results() {
		count++
	}

	NotEqual(t, count, 4)
}

func TestCancelStillEnqueing(t *testing.T) {

	pool := NewPool(2, 4)

	fn := func(job *Job) {

		i := job.Params()[0].(int)
		if i == 1 {
			job.Cancel()
			return
		}
		time.Sleep(time.Second * 1)
		job.Return(i)
	}

	for i := 0; i < 4; i++ {
		time.Sleep(200 * time.Millisecond)
		pool.Queue(fn, i)
	}

	var count int

	for range pool.Results() {
		count++
	}

	NotEqual(t, count, 4)
}

func TestPanicRecovery(t *testing.T) {

	pool := NewPool(2, 4)

	fn := func(job *Job) {

		i := job.Params()[0].(int)
		if i == 1 {
			panic("OMG OMG OMG! something bad happened!")
		}
		time.Sleep(time.Second * 1)
		job.Return(i)
	}

	for i := 0; i < 4; i++ {
		time.Sleep(200 * time.Millisecond)
		pool.Queue(fn, i)
	}

	var count int

	for result := range pool.Results() {
		err, ok := result.(*ErrRecovery)
		if ok {
			count++
			NotEqual(t, len(err.Error()), 0)
		}
	}

	Equal(t, count, 1)
}

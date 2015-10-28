package pool

import (
	"testing"
	"time"
)

func BenchmarkSmallRun(b *testing.B) {

	pool := NewPool(4, 10)

	fn := func(job *Job) {

		i := job.Params()[0].(int)
		time.Sleep(time.Second * 1)
		job.Return(i)
	}

	for i := 0; i < 10; i++ {
		pool.Queue(fn, i)
	}

	for range pool.Results() {
	}
}

func BenchmarkSmallCancel(b *testing.B) {

	pool := NewPool(4, 20)

	fn := func(job *Job) {

		i := job.Params()[0].(int)
		if i == 6 {
			job.Cancel()
			return
		}

		time.Sleep(time.Second * 1)
		job.Return(i)
	}

	for i := 0; i < 20; i++ {
		pool.Queue(fn, i)
	}

	for range pool.Results() {
	}
}

func BenchmarkLargeCancel(b *testing.B) {

	pool := NewPool(4, 1000)

	fn := func(job *Job) {

		i := job.Params()[0].(int)
		if i == 6 {
			job.Cancel()
			return
		}

		time.Sleep(time.Second * 1)
		job.Return(i)
	}

	for i := 0; i < 1000; i++ {
		pool.Queue(fn, i)
	}

	for range pool.Results() {
	}
}

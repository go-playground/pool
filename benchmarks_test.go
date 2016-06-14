package pool

import (
	"testing"
	"time"
)

// import (
// 	"testing"
// 	"time"
// )

func BenchmarkSmallRun(b *testing.B) {

	// var res []*WorkUnit
	res := make([]*WorkUnit, 10)

	// b.N = 30
	// b.ReportAllocs()

	// for n := 0; n < b.N; n++ {

	pool := New(10)
	defer pool.Close()

	fn := func() (interface{}, error) {
		time.Sleep(time.Second * 1)
		return 1, nil
	}

	// b.ReportAllocs()

	for i := 0; i < 10; i++ {
		// fmt.Println("Queue", i)
		res[i] = pool.Queue(fn)
		// res = append(res, pool.Queue(fn))
		// if i == 2 {
		// 	pool.Cancel()
		// 	break
		// }
		// fmt.Println("Continue")
	}

	var count int

	for _, cw := range res {
		// fmt.Println("Waiting for Done")
		if cw == nil {
			continue
		}

		<-cw.Done
		// fmt.Println("Done Waiting")

		if cw.Error == nil {
			count += cw.Value.(int)
		}
	}

	// if count != 10 {
	// 	b.Fatal("Count Incorrect")
	// }
	// }
}

// func BenchmarkSmallCancel(b *testing.B) {

// 	pool := NewPool(4, 20)

// 	fn := func(job *Job) {

// 		i := job.Params()[0].(int)
// 		if i == 6 {
// 			job.Cancel()
// 			return
// 		}

// 		time.Sleep(time.Second * 1)
// 		job.Return(i)
// 	}

// 	for i := 0; i < 20; i++ {
// 		pool.Queue(fn, i)
// 	}

// 	for range pool.Results() {
// 	}
// }

// func BenchmarkLargeCancel(b *testing.B) {

// 	pool := NewPool(4, 1000)

// 	fn := func(job *Job) {

// 		i := job.Params()[0].(int)
// 		if i == 6 {
// 			job.Cancel()
// 			return
// 		}

// 		time.Sleep(time.Second * 1)
// 		job.Return(i)
// 	}

// 	for i := 0; i < 1000; i++ {
// 		pool.Queue(fn, i)
// 	}

// 	for range pool.Results() {
// 	}
// }

// func BenchmarkOverconsumeLargeRun(b *testing.B) {

// 	pool := NewPool(25, 100)

// 	fn := func(job *Job) {

// 		i := job.Params()[0].(int)
// 		time.Sleep(time.Second * 1)
// 		job.Return(i)
// 	}

// 	for i := 0; i < 100; i++ {
// 		pool.Queue(fn, i)
// 	}

// 	for range pool.Results() {
// 	}
// }

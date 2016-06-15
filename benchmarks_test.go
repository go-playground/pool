package pool

import (
	"testing"
	"time"
)

func BenchmarkSmallRun(b *testing.B) {

	res := make([]*WorkUnit, 10)

	b.ReportAllocs()

	pool := New(10)
	defer pool.Close()

	fn := func() (interface{}, error) {
		time.Sleep(time.Second * 1)
		return 1, nil
	}

	for i := 0; i < 10; i++ {
		res[i] = pool.Queue(fn)
	}

	var count int

	for _, cw := range res {

		<-cw.Done

		if cw.Error == nil {
			count += cw.Value.(int)
		}
	}

	if count != 10 {
		b.Fatal("Count Incorrect")
	}
}

func BenchmarkSmallCancel(b *testing.B) {

	res := make([]*WorkUnit, 0, 20)

	b.ReportAllocs()

	pool := New(4)
	defer pool.Close()

	newFunc := func(i int) func() (interface{}, error) {
		return func() (interface{}, error) {
			time.Sleep(time.Second * 1)
			return i, nil
		}
	}

	for i := 0; i < 20; i++ {
		if i == 6 {
			pool.Cancel()
		}
		res = append(res, pool.Queue(newFunc(i)))
	}

	for _, wrk := range res {
		if wrk == nil {
			continue
		}
		<-wrk.Done
	}
}

func BenchmarkLargeCancel(b *testing.B) {

	res := make([]*WorkUnit, 0, 1000)

	b.ReportAllocs()

	pool := New(4)
	defer pool.Close()

	newFunc := func(i int) func() (interface{}, error) {
		return func() (interface{}, error) {
			time.Sleep(time.Second * 1)
			return i, nil
		}
	}

	for i := 0; i < 1000; i++ {
		if i == 6 {
			pool.Cancel()
		}
		res = append(res, pool.Queue(newFunc(i)))
	}

	for _, wrk := range res {
		if wrk == nil {
			continue
		}
		<-wrk.Done
	}
}

func BenchmarkOverconsumeLargeRun(b *testing.B) {

	res := make([]*WorkUnit, 100)

	b.ReportAllocs()

	pool := New(25)
	defer pool.Close()

	newFunc := func(i int) func() (interface{}, error) {
		return func() (interface{}, error) {
			time.Sleep(time.Second * 1)
			return 1, nil
		}
	}

	for i := 0; i < 100; i++ {
		res[i] = pool.Queue(newFunc(i))
	}

	var count int

	for _, cw := range res {

		<-cw.Done

		count += cw.Value.(int)
	}

	if count != 100 {
		b.Fatalf("Count Incorrect, Expected '100' Got '%d'", count)
	}
}

func BenchmarkBatchSmallRun(b *testing.B) {

	fn := func() (interface{}, error) {
		time.Sleep(time.Second * 1)
		return 1, nil
	}

	pool := New(10)
	defer pool.Close()

	batch := pool.Batch()

	for i := 0; i < 10; i++ {
		batch.Queue(fn)
	}

	batch.QueueComplete()

	var count int

	for cw := range batch.Results() {
		count += cw.Value.(int)
	}

	if count != 10 {
		b.Fatal("Count Incorrect")
	}
}

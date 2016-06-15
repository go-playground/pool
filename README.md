Package pool
============

[![Build Status](https://semaphoreci.com/api/v1/projects/a85ae32e-f437-40f1-9652-2525ec282658/593594/badge.svg)](https://semaphoreci.com/joeybloggs/pool)
[![Coverage Status](https://coveralls.io/repos/go-playground/pool/badge.svg?branch=v2&service=github)](https://coveralls.io/github/go-playground/pool?branch=v2)
[![GoDoc](https://godoc.org/gopkg.in/go-playground/pool.v2?status.svg)](https://godoc.org/gopkg.in/go-playground/pool.v2)

Package pool implements a consumer goroutine pool for easier goroutine handling. 

Features:

-    Dead simple to use and makes no assumptions about how you will use it.
-    Automatic recovery from consumer goroutines which returns an error to the results

Pool v2 advantages over Pool v1:

- Up to 300% faster due to lower contention ( BenchmarkSmallRun used to take 3 seconds, now 1 second )
- Cancels are much faster
- Easier to use, no longer need to know the # of Work Units to be processed.
- Pool can now be used as a long running/globally defined pool if desired ( v1 Pool was only good for one run )
- Supports single units of work as well as batching
- Pool can easily be reset after a Close() or Cancel() for reuse.
- Multiple Batches can be run and even cancelled on the same Pool.

Installation
------------

Use go get.

	go get gopkg.in/go-playground/pool.v2

Then import the pool package into your own code.

	import "gopkg.in/go-playground/pool.v2"


Important Information READ THIS!
------

- It is recommended that you cancel a pool or batch from the calling function and not inside of the Unit of Work, it will work fine, however because of the goroutine scheduler and context switching it may not cancel as soon as if called from outside.
- When Batching DO NOT FORGET TO CALL batch.QueueComplete(), if you do the Batch WILL deadlock

Usage and documentation
------

Please see http://godoc.org/gopkg.in/go-playground/pool.v2 for detailed usage docs.

##### Examples:

Per Unit Work
```go
package main

import (
	"fmt"
	"time"

	"gopkg.in/go-playground/pool.v2"
)

func main() {

	p := pool.New(10)
	defer p.Close()

	user := p.Queue(getUser(13))
	other := p.Queue(getOtherInfo(13))

	<-user.Done

	if user.Error != nil {
		// handle error
	}

	// do stuff with user
	username := user.Value.(string)
	fmt.Println(username)

	<-other.Done
	if other.Error != nil {
		// handle error
	}

	// do stuff with other
	otherInfo := other.Value.(string)
	fmt.Println(otherInfo)
}

func getUser(id int) pool.WorkFunc {
	return func() (interface{}, error) {
		time.Sleep(time.Second * 1)
		return "Joeybloggs", nil
	}
}

func getOtherInfo(id int) pool.WorkFunc {
	return func() (interface{}, error) {
		time.Sleep(time.Second * 1)
		return "Other Info", nil
	}
}
```

Batch Work
```go
package main

import (
	"time"

	"gopkg.in/go-playground/pool.v2"
)

func main() {

	p := pool.New(10)
	defer p.Close()

	batch := p.Batch()

	// for max speed Queue in another goroutine
	// but it is not required, just can't start reading results
	// until all items are Queued.

	go func() {
		for i := 0; i < 10; i++ {
			batch.Queue(sendEmail("email content"))
		}

		// DO NOT FORGET THIS OR GOROUTINES WILL DEADLOCK
		// if calling Cancel() it calles QueueComplete() internally
		batch.QueueComplete()
	}()

	for email := range batch.Results() {

		if email.Error != nil {
			// handle error
			// maybe call batch.Cancel()
		}
	}
}

func sendEmail(email string) pool.WorkFunc {
	return func() (interface{}, error) {
		time.Sleep(time.Second * 1)
		return nil, nil // everything ok, send nil, error if not
	}
}
```

Benchmarks
------
###### Run on MacBook Pro (Retina, 15-inch, Late 2013) 2.6 GHz Intel Core i7 16 GB 1600 MHz DDR3 using Go 1.6.2

run with 1, 2, 4,8 and 16 cpu to show it scales well..16 is double the # of logical cores on this machine.

NOTE: Cancellation times CAN vary depending how busy your system is and how the goroutine scheduler is but 
worse case I've seen is 1 second to cancel instead of 0ns

```go
go test -cpu=1,2,4,8,16 -bench=. -benchmem=true
PASS
BenchmarkSmallRun              	       1	1000769885 ns/op	    3568 B/op	      55 allocs/op
BenchmarkSmallRun-2            	       1	1000101971 ns/op	    3712 B/op	      58 allocs/op
BenchmarkSmallRun-4            	       1	1000149555 ns/op	    3776 B/op	      59 allocs/op
BenchmarkSmallRun-8            	       1	1001186229 ns/op	    4736 B/op	      74 allocs/op
BenchmarkSmallRun-16           	       1	1001276088 ns/op	    3008 B/op	      47 allocs/op
BenchmarkSmallCancel           	2000000000	         0.00 ns/op	       0 B/op	       0 allocs/op
BenchmarkSmallCancel-2         	2000000000	         0.00 ns/op	       0 B/op	       0 allocs/op
BenchmarkSmallCancel-4         	2000000000	         0.00 ns/op	       0 B/op	       0 allocs/op
BenchmarkSmallCancel-8         	2000000000	         0.00 ns/op	       0 B/op	       0 allocs/op
BenchmarkSmallCancel-16        	2000000000	         0.00 ns/op	       0 B/op	       0 allocs/op
BenchmarkLargeCancel           	2000000000	         0.00 ns/op	       0 B/op	       0 allocs/op
BenchmarkLargeCancel-2         	2000000000	         0.00 ns/op	       0 B/op	       0 allocs/op
BenchmarkLargeCancel-4         	2000000000	         0.00 ns/op	       0 B/op	       0 allocs/op
BenchmarkLargeCancel-8         	   10000	    100120 ns/op	      20 B/op	       0 allocs/op
BenchmarkLargeCancel-16        	2000000000	         0.00 ns/op	       0 B/op	       0 allocs/op
BenchmarkOverconsumeLargeRun   	       1	4004817389 ns/op	   27424 B/op	     433 allocs/op
BenchmarkOverconsumeLargeRun-2 	       1	4005885532 ns/op	   25632 B/op	     408 allocs/op
BenchmarkOverconsumeLargeRun-4 	       1	4003106218 ns/op	   28912 B/op	     459 allocs/op
BenchmarkOverconsumeLargeRun-8 	       1	4003867938 ns/op	   28736 B/op	     456 allocs/op
BenchmarkOverconsumeLargeRun-16	       1	4002726049 ns/op	   28848 B/op	     458 allocs/op
BenchmarkBatchSmallRun         	       1	1000501802 ns/op	    3456 B/op	      54 allocs/op
BenchmarkBatchSmallRun-2       	       1	1000147619 ns/op	    3456 B/op	      54 allocs/op
BenchmarkBatchSmallRun-4       	       1	1000496285 ns/op	    4048 B/op	      63 allocs/op
BenchmarkBatchSmallRun-8       	       1	1003713462 ns/op	    4368 B/op	      68 allocs/op
BenchmarkBatchSmallRun-16      	       1	1000396572 ns/op	    4160 B/op	      65 allocs/op
```
To put these benchmarks in perspective:

* BenchmarkSmallRun did 10 seconds worth of processing in 1.000769885 seconds
* BenchmarkSmallCancel ran 20 jobs, cancelled on job 6 and and ran in 0 seconds
* BenchmarkLargeCancel ran 1000 jobs, cancelled on job 6 and and ran in 0 seconds
* BenchmarkOverconsumeLargeRun ran 100 jobs using 25 workers in 4.004817389 seconds


License
------
Distributed under MIT License, please see license file in code for more details.

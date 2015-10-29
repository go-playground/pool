Library pool
============

[![GoDoc](https://godoc.org/github.com/joeybloggs/pool?status.svg)](https://godoc.org/github.com/joeybloggs/pool)

Library pool implements a consumer goroutine pool for easier goroutine handling. 

Features:

-    Dead simple to use and makes no assumptions about how you will use it.

Installation
------------

Use go get.

	go get github.com/joeybloggs/pool

or to update

	go get -u github.com/joeybloggs/pool

Then import the validator package into your own code.

	import "github.com/joeybloggs/pool"

Usage and documentation
------

Please see http://godoc.org/github.com/joeybloggs/pool for detailed usage docs.

##### Examples:

Struct return value
```go
package main

import (
	"errors"
	"fmt"
	"time"

	"github.com/joeybloggs/pool"
)

type resultStruct struct {
	i   int
	err error
}

func main() {
	p := pool.NewPool(4, 16)

	fn := func(job *pool.Job) {

		i := job.Params()[0].(int)

		res := &resultStruct{
			i: i,
		}

		// any condition that would cause an error
		if i == 10 {
			res.err = errors.New("Something bad happened, but don't need to cancel the rest of the jobs")
			job.Return(res)
			// or if you want to cancel run the line below
			job.Cancel()
			return
		}

		time.Sleep(time.Second * 1)
		job.Return(res)
	}

	for i := 0; i < 4; i++ {
		p.Queue(fn, i)
	}

	for result := range p.Results() {

		res := result.(*resultStruct)

		if res.err != nil {
			// do what you want with error or cancel the pool here p.Cancel()
			fmt.Println(res.err)
		}

		// do what you want with result
		fmt.Println(res.i)
	}
}
```

Value return value
```go
package main

import (
	"errors"
	"fmt"
	"time"

	"github.com/joeybloggs/pool"
)

func main() {
	p := pool.NewPool(4, 16)

	fn := func(job *pool.Job) {

		i := job.Params()[0].(int)

		// any condition that would cause an error
		if i == 10 {
			job.Return(errors.New("Something bad happened, but don't need to cancel the rest of the jobs"))
			// or if you want to cancel run the line below
			job.Cancel()
			return
		}

		time.Sleep(time.Second * 1)
		job.Return(i)
	}

	for i := 0; i < 4; i++ {
		p.Queue(fn, i)
	}

	for result := range p.Results() {
		switch result.(type) {
		case error:
			err := result.(error)
			// do what you want with error or cancel the pool here p.Cancel()
			fmt.Println(err)
		default:
			j := result.(int)
			// do what you want with result
			fmt.Println(j)
		}
	}
}
```

Benchmarks
------
###### Run on MacBook Pro (Retina, 15-inch, Late 2013) 2.6 GHz Intel Core i7 16 GB 1600 MHz DDR3 using Go 1.5.1
```go
$ go test -cpu=4 -bench=. -benchmem=true
PASS
BenchmarkSmallRun-4           	       1	3009120497 ns/op	    3360 B/op	      65 allocs/op
BenchmarkSmallCancel-4        	       1	2003173598 ns/op	    3696 B/op	      81 allocs/op
BenchmarkLargeCancel-4        	       1	2001222531 ns/op	  106784 B/op	    3028 allocs/op
BenchmarkOverconsumeLargeRun-4	       1	4004509778 ns/op	   36528 B/op	     661 allocs/op
ok  	github.com/joeybloggs/pool	14.230s
```
To put these benchmarks in perspective:

* BenchmarkSmallRun-4 did 10 seconds worth of processing in 3 seconds
* BenchmarkSmallCancel-4 ran 20 jobs, cancelled on job 6 and and ran in 2 seconds
* BenchmarkLargeCancel-4 ran 1000 jobs, cancelled on job 6 and and ran in 2 seconds
* BenchmarkOverconsumeLargeRun-4 ran 100 jobs using 25 consumers in 4 seconds

How to Contribute
------

There will always be a development branch for each version i.e. `v1-development`. In order to contribute, 
please make your pull requests against those branches.

If the changes being proposed or requested are breaking changes, please create an issue, for discussion
or create a pull request against the highest development branch for example this package has a
v1 and v1-development branch however, there will also be a v2-development branch even though v2 doesn't exist yet.

License
------
Distributed under MIT License, please see license file in code for more details.
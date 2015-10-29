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

		err, ok := result.(error)
		if ok {
			// there was some sort of panic that
			// was recovered, in this scenario
			fmt.Println(err)
			return
		}

		res := result.(*resultStruct)

		if res.err != nil {
			// do what you want with error or cancel the pool here p.Cancel()
			fmt.Println(res.err)
		}

		// do what you want with result
		fmt.Println(res.i)
	}
}

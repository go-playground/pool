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

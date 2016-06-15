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

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

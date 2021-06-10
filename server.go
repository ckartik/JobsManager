package main

import (
	"fmt"
	"time"

	Jobs "github.com/ckartik/jobsmanager/jobs"
)

func main() {
	jm := Jobs.JobsManager{}
	id, err := jm.Start("sleep", "2")
	if err != nil {
		fmt.Println(err)
	}
	time.Sleep(3 * time.Second)
	fmt.Println(id)
	ok, js := jm.Query(id)
	if ok {
		fmt.Println(*js)
	}
	Jobs.Test()
}

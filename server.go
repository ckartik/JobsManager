package main

import (
	"fmt"

	Jobs "github.com/ckartik/jobsmanager/jobs"
)

func main() {
	jm := Jobs.JobsManager{}
	id, err := jm.Start("sleep", "5")
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(id)
	ok, js := jm.Query(id)
	if ok {
		fmt.Println(*js)
	}
	Jobs.Test()
}

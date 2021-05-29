package main

import (
	"github.com/google/uuid"
	"os/exec"
	"sync"
	"fmt"
)

// Critical question is around how to select an identifier for each job.
// Pid is a bad choice as it cycles on the OS.
type jobsManager struct {
	jobsPool sync.Map //[uuid.UUID]string
	output sync.Map //[uuid.UUID][]byte	// What if the output of a job is too large.
}

type JobsManager interface {
	Start(string, ...string) (bool, uuid.UUID)
	Stop(uuid.UUID) (bool, error)
	Query(uuid.UUID) (string, error)
}

func NewJobsManager() JobsManager {
	return jobsManager{}
}

func (jm jobsManager) Start(cmd string, args ...string) (bool, uuid.UUID) {
	id := uuid.New()
	c := exec.Command(cmd)
	for _, arg := range args {
		c.Args = append(c.Args, arg)
	}
	err := c.Start()
	if err != nil {
		fmt.Printf("%v", err)
	}
	err = c.Wait()
	if err != nil {
		fmt.Printf("%v", err)
	}
	out, _ := c.Output()
	fmt.Printf("Output: %v", out)
	return true, id
}

func main() {
	jm := NewJobsManager()
	out, _ := jm.Start("wget", "www.google.com")
	newOut, _ := jm.Start("cat", "index.html")
	fmt.Printf("%v", out)
	fmt.Printf("\n newoutput:\n %v", newOut)
}
func (jm jobsManager) Stop(uuid.UUID) (bool, error) {
	_ = jm
	return true, nil
}

func (jm jobsManager) Query(uuid.UUID) (string, error) {
	_ = jm
	return "", nil
}

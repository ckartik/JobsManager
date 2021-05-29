package main

import (
	"github.com/google/uuid"
	"os/exec"
	"sync"
	"fmt"
	// "bytes"
)

// Critical question is around how to select an identifier for each job.
// Pid is a bad choice as it cycles on the OS.
type JobsManager struct {
	jobs sync.Map //[uuid.UUID]string
	output sync.Map //[uuid.UUID][]byte	// What if the output of a job is too large. - NOTE: Only read from this.
	wg sync.WaitGroup // Counts the number of workers currently running.
}

func (jm *JobsManager) Start(cmd string, args ...string) (bool, uuid.UUID) {
	id := uuid.New()
	// Construct Command
	c := exec.Command(cmd)
	for _, arg := range args {
		c.Args = append(c.Args, arg)
	}
	jm.jobs.Store(id, c)
	jm.wg.Add(1)
	go func(id uuid.UUID, c *exec.Cmd, wg *sync.WaitGroup){ 
		defer wg.Done()
		fmt.Println("Running Command")
		out, err := c.Output()
		if err != nil {
			fmt.Printf("Had the following issue: %v", err)
			return 
		}
		fmt.Println("storing value %v at id %v", out, id)
		jm.output.Store(id, out)
	}(id, c, &jm.wg)
	return true, id
}

func main() {
	jm := &JobsManager{}
	ok, id := jm.Start("wget", "www.google.com")
	if !ok {
		fmt.Println("Job not started")
	}
	ok, id2 := jm.Start("cat", "index.html")
	if !ok {
		fmt.Println("Job not started")
	}
	jm.wg.Wait()
	out, _ := jm.output.Load(id)
	fmt.Printf("%s", out)
	newOut, _ := jm.output.Load(id2)
	fmt.Printf("\n newoutput:\n %s", newOut)
}
func (jm *JobsManager) Stop(uuid.UUID) (bool, error) {
	_ = jm
	return true, nil
}

func (jm *JobsManager) Query(uuid.UUID) (string, error) {
	_ = jm
	return "", nil
}

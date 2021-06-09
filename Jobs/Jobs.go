package Jobs

import (
	"fmt"
	"os/exec"
	"sync"

	"github.com/google/uuid"
)

type JobStatus struct {
	// Options - ["Stopped", "Completed", "Errored"]
	status   string
	exitCode uint8
	output   JobOutput
}

type JobOutput struct {
	StdOut []byte
	StdErr []byte
}

type JobsInfo struct {
	Command   *exec.Cmd
	JobStatus JobStatus
}

// Note: All channels will be buffered with cap 1.
type JobChans struct {
	// Write from API
	Kill chan struct{}
	// Read from API
	Status chan JobStatus
}

type JobsManager struct {
	JobInfo     sync.Map // uuid -> JobsInfo
	JobChannels sync.Map
}

func (jm *JobsManager) Start(cmd string, args ...string) (uuid.UUID) {
	id := uuid.New()
	return id
}

func (jm *JobsManager) Stop(id uuid.UUID) (bool, error) {
	_ = id
	return true, nil
}

func (jm *JobsManager) Query(id uuid.UUID) (bool, JobStatus) {
	return true, JobStatus{}
}

func Test() {
	fmt.Println("HELLLOO")
}
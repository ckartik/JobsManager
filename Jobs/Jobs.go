package Jobs

import (
	"os/exec"
	"sync"
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

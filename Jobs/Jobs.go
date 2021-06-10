package Jobs

import (
	"fmt"
	"io"
	"log"
	"os/exec"
	"sync"

	"github.com/google/uuid"
)

type Status string

const (
	Completed Status = "Completed"
	Errored   Status = "Errored"
	Stopped   Status = "Stopped"
	Running   Status = "Running"
)

type JobStatus struct {
	State    Status
	ExitCode int
	Output   *JobOutput
}

type JobOutput struct {
	StdOut []byte
	StdErr []byte
}

type JobInfo struct {
	Command   *exec.Cmd
	JobStatus *JobStatus
}

type JobChans struct {
	// Write from API
	Kill chan struct{}
	// Read from API
	Status chan JobStatus
}

type JobsManager struct {
	JobInfos    sync.Map // uuid -> JobInfo
	JobChannels sync.Map
}

func initJobWorker(id uuid.UUID, info JobInfo, channels JobChans) error {
	output := new(JobOutput)
	stderr, err := info.Command.StderrPipe()
	if err != nil {
		log.Fatal(err)
	}
	stdout, err := info.Command.StdoutPipe()
	if err != nil {
		log.Fatal(err)
	}

	// TODO(@ckartik): Ensure that if command errors out early, we highlight in status/query.
	err = info.Command.Start()
	go func(output *JobOutput, channels JobChans, info JobInfo) {
		jobStatus := JobStatus{State: Running, ExitCode: -1, Output: nil}

		output.StdErr, err = io.ReadAll(stderr)
		output.StdOut, err = io.ReadAll(stdout)

		if err := info.Command.Wait(); err != nil {
			select {
			case <-channels.Kill:
				jobStatus.State = Stopped
			default:
				jobStatus.State = Errored
			}
			jobStatus.ExitCode = err.(*exec.ExitError).ExitCode()
		} else {

		}

	}(output, channels, info)
	return err
}

func (jm *JobsManager) Start(cmd string, args ...string) (uuid.UUID, error) {
	id := uuid.New()
	c := exec.Command(cmd, args...)
	info := JobInfo{Command: c, JobStatus: nil}

	killChannel := make(chan struct{}, 1)
	statusChan := make(chan JobStatus, 1)
	jobChans := JobChans{Kill: killChannel, Status: statusChan}
	jm.JobChannels.Store(id, jobChans)

	err := initJobWorker(id, info, jobChans)
	if err != nil {
		return id, err
	}

	return id, nil
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

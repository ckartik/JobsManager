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
	Status chan *JobStatus
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
		var jobStatus *JobStatus

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
			jobStatus.ExitCode = 0
		}

		channels.Status <- jobStatus
	}(output, channels, info)

	return err
}

func (jm *JobsManager) Start(cmd string, args ...string) (uuid.UUID, error) {
	id := uuid.New()
	c := exec.Command(cmd, args...)

	killChannel := make(chan struct{}, 1)
	statusChan := make(chan *JobStatus, 1)
	jobChans := JobChans{Kill: killChannel, Status: statusChan}
	jm.JobChannels.Store(id, jobChans)

	info := JobInfo{Command: c, JobStatus: nil}
	jm.JobInfos.Store(id, info)

	if err := initJobWorker(id, info, jobChans); err != nil {
		return id, err
	}

	return id, nil
}

func (jm *JobsManager) Stop(id uuid.UUID) (bool, error) {
	_ = id
	return true, nil
}

func (jm *JobsManager) Query(id uuid.UUID) (bool, *JobStatus) {
	if chans, ok := jm.JobChannels.Load(id); ok {
		select {
		case status := <-chans.(JobChans).Status:
			info, _ := jm.JobInfos.Load(id)
			info.(*JobInfo).JobStatus = status
			jm.JobInfos.Store(id, info)
		default:
		}

		info, _ := jm.JobInfos.Load(id)
		return true, info.(*JobInfo).JobStatus
	}

	return false, nil
}

func Test() {
	fmt.Println("HELLLOO")
}

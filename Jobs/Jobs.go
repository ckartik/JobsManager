package Jobs

import (
	"errors"
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
	NotFound  Status = "Not Found"
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
	Command *exec.Cmd
	Status  JobStatus
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
		var jobStatus JobStatus

		output.StdErr, err = io.ReadAll(stderr)
		output.StdOut, err = io.ReadAll(stdout)
		jobStatus.Output = output
		if err := info.Command.Wait(); err != nil {
			select {
			case <-channels.Kill:
				jobStatus.State = Stopped
			default:
				jobStatus.State = Errored
			}
			jobStatus.ExitCode = err.(*exec.ExitError).ExitCode()
		} else {
			jobStatus = JobStatus{State: Completed, ExitCode: 0, Output: output}
		}

		channels.Status <- jobStatus
	}(output, channels, info)

	return err
}

func (jm *JobsManager) Start(cmd string, args ...string) (uuid.UUID, error) {
	id := uuid.New()
	c := exec.Command(cmd, args...)

	killChannel := make(chan struct{}, 1)
	statusChan := make(chan JobStatus, 1)
	jobChans := JobChans{Kill: killChannel, Status: statusChan}
	jm.JobChannels.Store(id, jobChans)

	status := JobStatus{State: Running, ExitCode: -1, Output: nil}
	info := JobInfo{Command: c, Status: status}
	jm.JobInfos.Store(id, info)

	if err := initJobWorker(id, info, jobChans); err != nil {
		return id, err
	}

	return id, nil
}

func (jm *JobsManager) Stop(id uuid.UUID) (bool, error) {
	if info, ok := jm.JobInfos.Load(id); ok {
		if info.(JobInfo).Status.State == Running {
			chans, _ := jm.JobChannels.Load(id)
			chans.(JobChans).Kill <- struct{}{}
			err := info.(JobInfo).Command.Process.Kill()
			if err != nil {
				return false, err
			}

			return true, nil
		}
	}

	return false, errors.New("ID not set")
}

func (jm *JobsManager) Query(id uuid.UUID) (bool, *JobStatus) {
	if chans, ok := jm.JobChannels.Load(id); ok {
		var info JobInfo
		select {
		case status := <-chans.(JobChans).Status:
			uncastedInfo, _ := jm.JobInfos.Load(id)
			info = uncastedInfo.(JobInfo)
			info.Status = status
			jm.JobInfos.Store(id, info)
		default:
			uncastedInfo, _ := jm.JobInfos.Load(id)
			info = uncastedInfo.(JobInfo)
		}

		return true, &(info.Status)
	}

	return false, nil
}

func Test() {
	fmt.Println("HELLLOO")
}

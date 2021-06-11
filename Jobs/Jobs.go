// Package jobs implemnts methods for interacting with OS manage commands called "Jobs".
package jobs

import (
	"errors"
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

// initJobWorker is the primary worker of the JobsService. It will coordinate the
// running of a job, it's closing, output handling, and coordinate setting state,
// to reflect erroed v.s user stopped.
// If unable to intialize the command, it will return an error.
//
// the method does not wait for processing to compelete, but rather spins up a
// new goroutine.
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
		// Note: we ensure that output is set and in a immutable state before communicating
		// 	 it back through the channel, this allows us to maintain it as a pointer.
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

// Start sets up the command and initializes it.
// It returns a probablistically unique uuidv4, which can be used to stop/query.
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

// Stop sets up state to be set to Kill and sends a KILL SIG to the process.
// If Stop can't find the corresponding `id` it will return false, error.
func (jm *JobsManager) Stop(id uuid.UUID) (bool, error) {
	if info, ok := jm.JobInfos.Load(id); ok {
		if info.(JobInfo).Status.State == Running {
			chans, _ := jm.JobChannels.Load(id)
			// Note: Trick to avoid blocking.
			select {
			case chans.(JobChans).Kill <- struct{}{}:
			default:
			}
			err := info.(JobInfo).Command.Process.Kill()
			if err != nil {
				return false, err
			}

			return true, nil
		}
	}

	return false, errors.New("ID not set")
}

// Query will retrieve the current [JobStatus] of the job that corresponds with `id`.
// If a job with `id` cannot be found, the function will return false, nil.
// On success, it returns true, and a pointer to the jobstatus auto-stored in the heap.
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
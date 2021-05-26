package jobs

import "sync"

type jobsManger struct {
	jobsPool sync.Map
}

type JobsManager interface {
	Start(string) (bool, int)
	Stop(int) (bool, error)
	Query(int) (string, error)
}

func NewJobsManager() JobsManager {
	return jobsManger{}
}

func (jm jobsManager) Start(string) (bool, int) {
	_ = jm
	return true, 0

}

func (jm jobsManager) Stop(string) (bool, error) {
	_ = jm
	return true, nil
}

func (jm jobsManager) Query(string) (string, error) {
	_ = jm
	return "", nil
}
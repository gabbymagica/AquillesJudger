package services

import (
	"IFJudger/pkg/worker"
)

type WorkerService struct{}

func StartWorkerService() (*WorkerService, error) {
	return &WorkerService{}, nil
}

func (s *WorkerService) ExecuteWorker(code, input string) (string, string, error) {
	worker, err := worker.NewWorker()
	if err != nil {
		return "", "", err
	}

	worker.SetupPython(64)
	stdout, stderr, err := worker.Execute(code, input, 5)
	if err != nil {
		return stdout, stderr, err
	}

	return stdout, stderr, err
}

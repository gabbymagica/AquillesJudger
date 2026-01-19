package services

import (
	"IFJudger/internal/models"
	"IFJudger/internal/models/configs"
)

type JudgerService struct {
	workerService *WorkerService
	configJudger  configs.ConfigJudger
}

func StartJudgerService(workerService *WorkerService, configJudger configs.ConfigJudger) (*JudgerService, error) {
	return &JudgerService{
		workerService: workerService,
		configJudger:  configJudger,
	}, nil
}

func (s *JudgerService) EnqueueJudge(problemUseCases models.ProblemUseCasesLimits) {

}

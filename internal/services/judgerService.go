package services

import (
	"IFJudger/internal/api/dto"
	"IFJudger/internal/models"
	"fmt"
	"time"
)

type JudgerService struct {
	workerService *WorkerService
	cacheService  *CacheService
}

func StartJudgerService(workerService *WorkerService, cacheService *CacheService) (*JudgerService, error) {
	return &JudgerService{
		workerService: workerService,
		cacheService:  cacheService,
	}, nil
}

func (s *JudgerService) EnqueueJudge(judgeRequest dto.JudgeRequest) (string, error) {
	limits, path, err := s.cacheService.GetProblemData(judgeRequest.ProblemID)
	if err != nil {
		return "", err
	}

	limit, err := FindLimitToken(judgeRequest.LanguageToken, &limits)
	if err != nil {
		return "", err
	}

	token, err := LanguageTokenToID(judgeRequest.LanguageToken)
	if err != nil {
		return "", err
	}

	job := models.Job{
		LanguageID:   token,
		CachePath:    path,
		TimeLimit:    time.Duration(limit.TimeLimitSeconds * int(time.Second)),
		MaximumRamMB: limit.MaximumRamMB,
		Code:         judgeRequest.Code,
	}

	id, err := s.workerService.EnqueueJob(job)
	if err != nil {
		return "", nil
	}
	return id, nil
}

func (s *JudgerService) GetResult(token string) (models.JobResult, error) {
	result, exists := s.workerService.GetResult(token)
	if exists == false {
		return models.JobResult{}, fmt.Errorf("job does not exist")
	}

	return result, nil
}

func LanguageTokenToID(token string) (models.LanguageID, error) {
	if token == "python" {
		return models.Python, nil
	} else {
		return 0, fmt.Errorf("invalid language")
	}
}

func FindLimitToken(token string, limits *[]models.LanguageLimits) (*models.LanguageLimits, error) {
	for _, limit := range *limits {
		if limit.Name == token {
			return &limit, nil
		}
	}

	return nil, fmt.Errorf("limit not found for token %s in problem", token)
}

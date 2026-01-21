package services

import (
	"IFJudger/internal/models"
	"IFJudger/internal/models/configs"
	"IFJudger/pkg/worker"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"sync"
)

type WorkerService struct {
	config configs.WorkerServiceConfig

	jobQueue   chan models.Job
	results    sync.Map
	maxWorkers int
}

var LanguageNotFound = errors.New("language not found")

func StartWorkerService(config configs.WorkerServiceConfig) (*WorkerService, error) {
	log.Printf("[Init] Iniciando WorkerService com %d workers e fila de tamanho %d\n", config.MaxWorkers, config.QueueSize)

	service := &WorkerService{
		config:     config,
		jobQueue:   make(chan models.Job, config.QueueSize),
		results:    sync.Map{},
		maxWorkers: config.MaxWorkers,
	}

	service.startWorkers()

	return service, nil
}

func (s *WorkerService) startWorkers() {
	for i := 0; i < s.maxWorkers; i++ {
		go s.workerLoop(i)
	}
}

func (s *WorkerService) workerLoop(workerID int) {
	log.Printf("[Worker-%d] Pronto e aguardando jobs...\n", workerID)
	for {
		job, isOpen := <-s.jobQueue
		if !isOpen {
			log.Printf("[Worker-%d] Canal fechado. Encerrando.\n", workerID)
			break
		}

		log.Printf("[Worker-%d] Pegou o Job %s. Processando...\n", workerID, job.ID)
		s.processJob(job, workerID)
		log.Printf("[Worker-%d] Terminou o Job %s. Voltando a dormir.\n", workerID, job.ID)
	}
}

func (s *WorkerService) processJob(job models.Job, workerID int) {
	s.updateResult(job.ID, models.StatusProcessing, models.ExecutionReport{}, "")

	result, err := s.executeWorker(job, workerID)

	if err != nil {
		log.Printf("[Worker-%d] ERRO no Job %s: %v\n", workerID, job.ID, err)
		s.updateResult(job.ID, models.StatusError, models.ExecutionReport{}, err.Error())
		return
	}

	log.Printf("[Worker-%d] SUCESSO no Job %s\n", workerID, job.ID)
	s.updateResult(job.ID, models.StatusSuccess, result, "")
}

func (s *WorkerService) executeWorker(job models.Job, workerID int) (models.ExecutionReport, error) {
	log.Printf("[Worker-%d] -> Criando container Docker (RAM: %dMB, Timeout: %s)...\n", workerID, job.MaximumRamMB, job.TimeLimit)

	w, err := worker.NewWorker(worker.WorkerConfigData{
		ContainerTimeout: s.config.ContainerTimeout,
		TestTimeout:      job.TimeLimit,
		MaximumRamMB:     job.MaximumRamMB,
	})
	if err != nil {
		return models.ExecutionReport{}, fmt.Errorf("falha newWorker: %w", err)
	}
	defer w.Cleanup()

	log.Printf("[Worker-%d] -> Preparando Workspace em %s...\n", workerID, s.config.ExecutionDirectory)
	err = w.PrepareWorkspace(worker.DockerWorkspaceConfig{
		CachePath:          job.CachePath,
		ExecutionDirectory: s.config.ExecutionDirectory,
		RunnerPath:         s.config.RunnerPath,
	})
	if err != nil {
		return models.ExecutionReport{}, fmt.Errorf("falha prepareWorkspace: %w", err)
	}

	if job.LanguageID == models.Python {
		log.Printf("[Worker-%d] -> Configurando Python...\n", workerID)
		err = w.SetupPython(job.Code)
		if err != nil {
			return models.ExecutionReport{}, fmt.Errorf("falha setupPython: %w", err)
		}
	} else {
		return models.ExecutionReport{}, fmt.Errorf("invalid language ID: %v", job.LanguageID)
	}

	log.Printf("[Worker-%d] -> Executando Container...\n", workerID)
	workerResult, err := w.Execute()
	if err != nil {
		return models.ExecutionReport{}, fmt.Errorf("falha execute: %w", err)
	}

	log.Printf("[Worker-%d] -> Container finalizado. Resultado lido.\n", workerID)

	return mapToDomainReport(workerResult), nil
}

func mapToDomainReport(wr worker.ExecutionReport) models.ExecutionReport {
	domainResults := make([]models.TestCaseResult, len(wr.Results))

	for i, res := range wr.Results {
		domainResults[i] = models.TestCaseResult{
			ID:      res.ID,
			Status:  res.Status,
			TimeMS:  res.TimeMS,
			Message: res.Message,
		}
	}

	return models.ExecutionReport{
		Results: domainResults,
	}
}

func (s *WorkerService) EnqueueJob(job models.Job) (string, error) {
	jobID := generateToken()
	job.ID = jobID

	log.Printf("[API] Tentando enfileirar Job %s...\n", jobID)

	select {
	case s.jobQueue <- job:
		log.Printf("[API] Job %s entrou na fila.\n", jobID)
		s.updateResult(jobID, models.StatusQueued, models.ExecutionReport{}, "")
		return jobID, nil

	default:
		log.Printf("[API] WARN: Fila cheia! Rejeitando Job %s.\n", jobID)
		return "", fmt.Errorf("server is busy (queue full)")
	}
}

func generateToken() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func (s *WorkerService) updateResult(token, status string, result models.ExecutionReport, err string) {
	s.results.Store(token, models.JobResult{
		ID:           token,
		Status:       status,
		Result:       result,
		ErrorMessage: err,
	})
}

func (s *WorkerService) GetResult(token string) (models.JobResult, bool) {
	result, ok := s.results.Load(token)
	if !ok {
		return models.JobResult{}, false
	}
	return result.(models.JobResult), true
}

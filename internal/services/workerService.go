package services

import (
	"IFJudger/internal/models"
	"IFJudger/internal/models/configs"
	"IFJudger/internal/repository"
	"IFJudger/pkg/worker"
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

type WorkerService struct {
	repository *repository.SubmissionRepository

	config configs.WorkerServiceConfig

	jobQueue   chan models.Job
	maxWorkers int
}

var LanguageNotFound = errors.New("language not found")

func StartWorkerService(config configs.WorkerServiceConfig, repository *repository.SubmissionRepository) (*WorkerService, error) {
	log.Printf("[Init] Iniciando WorkerService com %d workers e fila de tamanho %d\n", config.MaxWorkers, config.QueueSize)

	service := &WorkerService{
		repository: repository,
		config:     config,
		jobQueue:   make(chan models.Job, config.QueueSize),
		maxWorkers: config.MaxWorkers,
	}

	service.cleanupStaleWorkspaces()

	service.startWorkers()

	go service.recoverJobs()

	return service, nil
}

func (s *WorkerService) cleanupStaleWorkspaces() {
	dir := s.config.ExecutionDirectory
	log.Printf("[Cleanup] Verificando lixo em %s...\n", dir)

	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return
		}
		log.Printf("[Cleanup] Erro ao ler diretório: %v\n", err)
		return
	}

	count := 0
	for _, e := range entries {
		if e.IsDir() && strings.HasPrefix(e.Name(), "job-") {
			fullPath := filepath.Join(dir, e.Name())
			if err := os.RemoveAll(fullPath); err != nil {
				log.Printf("[Cleanup] Falha ao remover %s: %v\n", e.Name(), err)
			} else {
				count++
			}
		}
	}
	if count > 0 {
		log.Printf("[Cleanup] %d pastas temporárias antigas removidas.\n", count)
	}
}

func (s *WorkerService) recoverJobs() {
	log.Println("[Recovery] Verificando jobs pendentes no banco...")

	jobs, err := s.repository.GetRecoverableJobs()
	if err != nil {
		log.Printf("[Recovery] Erro ao buscar jobs: %v\n", err)
		return
	}

	if len(jobs) == 0 {
		log.Println("[Recovery] Nenhum job pendente encontrado.")
		return
	}

	log.Printf("[Recovery] %d jobs encontrados. Re-enfileirando...\n", len(jobs))

	for _, job := range jobs {
		s.jobQueue <- job
		log.Printf("[Recovery] Job %s recuperado e re-enfileirado.\n", job.ID)
	}
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

	jobResult, _ := s.GetResult(job.ID)
	go s.sendCallback(&jobResult)
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

	if err := s.repository.CreateJob(job); err != nil {
		log.Printf("[API] ERRO CRÍTICO: Falha ao salvar job no banco: %v\n", err)
		return "", fmt.Errorf("database error")
	}

	select {
	case s.jobQueue <- job:
		log.Printf("[API] Job %s entrou na fila.\n", jobID)
		return jobID, nil

	default:
		log.Printf("[API] WARN: Fila cheia! Rejeitando Job %s.\n", jobID)

		s.updateResult(jobID, models.StatusError, models.ExecutionReport{}, "Job Rejected, queue is full")
		return "", fmt.Errorf("server is busy (queue full)")
	}
}

func generateToken() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func (s *WorkerService) updateResult(token, status string, result models.ExecutionReport, err string) {
	dbErr := s.repository.UpdateResult(models.JobResult{
		ID:           token,
		Status:       status,
		Result:       result,
		ErrorMessage: err,
	})

	if dbErr != nil {
		log.Printf("[ERROR] Falha ao atualizar job %s no banco: %v", token, dbErr)
	}
}

func (s *WorkerService) GetResult(token string) (models.JobResult, bool) {
	result, err := s.repository.GetByID(token)
	if err != nil {
		return models.JobResult{}, false
	}
	return result, true
}

func (s *WorkerService) sendCallback(result *models.JobResult) {
	jsonData, err := json.Marshal(result)
	if err != nil {
		fmt.Printf("Erro ao serializar: %v\n", err)
		return
	}

	resp, err := http.Post(s.config.CallbackUrl, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Printf("Erro na requisição: %v\n", err)
		return
	}

	defer resp.Body.Close()
}

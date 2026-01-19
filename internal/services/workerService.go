package services

import (
	"IFJudger/internal/models"
	"IFJudger/pkg/worker"
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"sync"
	"time"
)

type WorkerService struct {
	jobQueue   chan models.Job // channel pra receber jobs
	results    sync.Map        // hashmap para guardar os resultados (sync.Map é thread-safe)
	maxWorkers int             // máximo de workers, dockers rodando
}

var LanguageNotFound = errors.New("language not found")

func StartWorkerService() (*WorkerService, error) {
	service := &WorkerService{
		jobQueue:   make(chan models.Job, 3), // 3 (todo: tem que vir do .env)
		results:    sync.Map{},
		maxWorkers: 3, // 3 (todo: tem que vir do .env)
	}

	service.startWorkers()

	return service, nil
}

func (s *WorkerService) startWorkers() {
	for i := 0; i < s.maxWorkers; i++ {
		go s.workerLoop(i) // inicializa funções anônimas para cada worker
	}
}

func (s *WorkerService) workerLoop(workerID int) {
	log.Printf("[Worker %d] Iniciado e esperando jobs...\n", workerID)
	for {
		job, isOpen := <-s.jobQueue // bloqueante
		if !isOpen {                // isopen vai ser false só quando fecharmos o worker
			break
		}

		log.Printf("[Worker %d] 🟢 Pegou o Job %s da fila\n", workerID, job.ID)
		start := time.Now()
		s.processJob(job, workerID)
		duration := time.Since(start)

		log.Printf("[Worker %d] 🏁 Finalizou Job %s em %v\n", workerID, job.ID, duration)
		if job.WebhookURL != "" {
			result, _ := s.GetResult(job.ID)

			go s.sendWebhook(job.WebhookURL, result)
		}
	}
}

func (s *WorkerService) processJob(job models.Job, workerID int) {
	s.updateResult(job.ID, "processing", "", "", "")
	log.Printf("[Worker %d] 🐳 Executando Docker para Job %s...\n", workerID, job.ID)

	stdout, stderr, err := s.executeWorker(job)
	if err != nil {
		log.Printf("[Worker %d] ❌ Erro no Job %s: %v\n", workerID, job.ID, err)
		s.updateResult(job.ID, "error", "", "", err.Error())
		return
	}

	s.updateResult(job.ID, "success", stdout, stderr, "")
}

func (s *WorkerService) executeWorker(job models.Job) (string, string, error) {
	worker, err := worker.NewWorker()
	if err != nil {
		return "", "", err
	}

	switch job.LanguageID {
	case 1:
		worker.SetupPython(int64(job.MaximumRamMB))
	default:
		return "", "", LanguageNotFound
	}

	stdout, stderr, err := worker.Execute(job.Code, job.Input, time.Duration(job.TimeLimitSeconds))
	if err != nil {
		return stdout, stderr, err
	}

	return stdout, stderr, err
}

func (s *WorkerService) sendWebhook(url string, result models.JobResult) {
	jsonData, err := json.Marshal(result)
	if err != nil {
		log.Printf("[Webhook] ❌ Erro ao criar JSON para %s: %v", url, err)
		return
	}

	// cria a requisição
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		log.Printf("[Webhook] ❌ Erro ao criar request: %v", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")

	// cria o cliente HTTP com timeout para não ficar preso eternamente
	client := &http.Client{Timeout: 10 * time.Second}

	// cliente, faça a requisição
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("[Webhook] ❌ Falha ao entregar em %s: %v", url, err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		log.Printf("[Webhook] ✅ Entregue com sucesso em %s", url)
	} else {
		log.Printf("[Webhook] ⚠️ Cliente retornou status %d", resp.StatusCode)
	}
}

// leitura e escrita do map de resultados
func (s *WorkerService) updateResult(token, status, out, errOut, execErr string) {
	s.results.Store(token, models.JobResult{
		ID:     token,
		Status: status,
		Stdout: out,
		Stderr: errOut,
		Error:  execErr,
	})
}

// cria o job, coloca na fila e retorna o token dele
func (s *WorkerService) EnqueueJob(job models.Job, webhookURL string) string {
	jobID := generateToken()

	s.updateResult(jobID, "queued", "", "", "")

	log.Printf("[API] 📥 Tentando enfileirar Job %s. Fila atual: %d/%d\n", jobID, len(s.jobQueue), cap(s.jobQueue))
	s.jobQueue <- job
	log.Printf("[API] ✅ Job %s entrou no buffer.\n", jobID)

	return jobID
}

// pega resultado no map
func (s *WorkerService) GetResult(token string) (models.JobResult, bool) {
	result, ok := s.results.Load(token)
	if !ok {
		return models.JobResult{}, false
	}
	return result.(models.JobResult), true
}

// gera token aleatório
func generateToken() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func tokenToIDLanguage(token string) {

}

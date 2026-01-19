package controllers

import (
	"IFJudger/internal/services"
	"net/http"
)

type WorkerController struct {
	WorkerService *services.WorkerService
}

func StartWorkerController(workerService *services.WorkerService) (*WorkerController, error) {
	return &WorkerController{WorkerService: workerService}, nil
}

// post route
func (c *WorkerController) HandleExecution(w http.ResponseWriter, r *http.Request) {
	panic("not implemented")
	/*var requestDTO dto.ExecutionRequest

	err := json.NewDecoder(r.Body).Decode(&requestDTO)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	defer r.Body.Close()

	token := c.WorkerService.EnqueueJob(requestDTO.Code, requestDTO.Input)

	executionEnqueuedResponse := dto.ExecutionEnqueuedResponse{
		Token:   token,
		Message: "Execution enqueued",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(executionEnqueuedResponse)*/
}

func (c *WorkerController) HandleStatus(w http.ResponseWriter, r *http.Request) {
	panic("not implemented")
	/*token := r.URL.Query().Get("token")

	jobResult, ok := c.WorkerService.GetResult(token)
	if !ok {
		http.Error(w, "Invalid token", http.StatusBadRequest)
		return
	}

	statusResponse := dto.ExecutionResponse{
		ID:     jobResult.ID,
		Status: jobResult.Status,
		Stdout: jobResult.Stdout,
		Stderr: jobResult.Stderr,
		Error:  jobResult.Error,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(statusResponse)*/
}

package controllers

import (
	dto "IFJudger/internal/models"
	"IFJudger/internal/services"
	"encoding/json"
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
	var requestDTO dto.ExecutionRequest

	err := json.NewDecoder(r.Body).Decode(&requestDTO)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	defer r.Body.Close()

	stdout, stderr, execErr := c.WorkerService.ExecuteWorker(requestDTO.Code, requestDTO.Input)

	responseDTO := dto.ExecutionResponse{
		Stdout: stdout,
		Stderr: stderr,
		//Error:  execErr.Error(),
	}

	if execErr != nil {
		responseDTO.Error = execErr.Error()
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(responseDTO)
}

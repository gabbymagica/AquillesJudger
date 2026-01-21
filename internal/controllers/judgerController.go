package controllers

import (
	"IFJudger/internal/api/dto"
	"IFJudger/internal/services"
	"encoding/json"
	"net/http"
)

type JudgerController struct {
	judgerService *services.JudgerService
}

func StartJudgerController(judgerService *services.JudgerService) (*JudgerController, error) {
	return &JudgerController{
		judgerService: judgerService,
	}, nil
}

func (c *JudgerController) HandleSubmission(w http.ResponseWriter, r *http.Request) {
	var req dto.SubmissionRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON format", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	if req.ProblemID == "" || req.Code == "" || req.Language == "" {
		http.Error(w, "Missing required fields (problem_id, code, language)", http.StatusBadRequest)
		return
	}

	serviceRequest := dto.JudgeRequest{
		ProblemID:     req.ProblemID,
		LanguageToken: req.Language,
		Code:          req.Code,
	}

	token, err := c.judgerService.EnqueueJudge(serviceRequest)
	if err != nil {
		http.Error(w, "Failed to enqueue submission: "+err.Error(), http.StatusInternalServerError)
		return
	}

	response := dto.SubmissionResponseDTO{
		Token:   token,
		Message: "Submission enqueued successfully",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

func (c *JudgerController) HandleStatus(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		http.Error(w, "Missing 'token' query parameter", http.StatusBadRequest)
		return
	}

	jobResult, err := c.judgerService.GetResult(token)
	if err != nil {
		if err.Error() == "job does not exist" {
			http.Error(w, "Submission not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	response := dto.StatusResponseDTO{
		ID:           jobResult.ID,
		Status:       jobResult.Status,
		Result:       jobResult.Result,
		ErrorMessage: jobResult.ErrorMessage,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

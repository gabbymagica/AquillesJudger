package dto

type SubmissionRequestDTO struct {
	ProblemID string `json:"problem_id"`
	Language  string `json:"language"`
	Code      string `json:"code"`
}

type SubmissionResponseDTO struct {
	Token   string `json:"token"`
	Message string `json:"message"`
}

type StatusResponseDTO struct {
	ID           string      `json:"id"`
	Status       string      `json:"status"`
	Result       interface{} `json:"result,omitempty"`
	ErrorMessage string      `json:"error,omitempty"`
}

package dto

type JudgeRequest struct {
	ProblemID     string `json:"problem_id"`
	LanguageToken string `json:"language_token"`
	Code          string `json:"code"`
}

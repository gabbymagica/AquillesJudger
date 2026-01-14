package dto

type JudgeRequest struct {
	ProblemID  int    `json:"problem_id"`
	LanguageID int    `json:"language_id"`
	Code       string `json:"code"`
}

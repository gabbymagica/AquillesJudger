package dto

type ExecutionRequest struct {
	Code  string `json:"code"`
	Input string `json:"input"`
}

type ExecutionResponse struct {
	Stdout string `json:"stdout"`
	Stderr string `json:"stderr"`
	Error  string `json:"error"`
}

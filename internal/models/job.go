package models

type Job struct {
	ID               string
	Code             string
	Input            string
	LanguageID       int
	TimeLimitSeconds int
	MaximumRamMB     int
	WebhookURL       string
}

type JobResult struct {
	ID     string
	Status string // "queued", "processing", "success", "error"
	Stdout string
	Stderr string
	Error  string
}

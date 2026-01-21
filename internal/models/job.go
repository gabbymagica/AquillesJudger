package models

import (
	"time"
)

type Job struct {
	ID           string
	LanguageID   LanguageID
	CachePath    string
	TimeLimit    time.Duration
	MaximumRamMB int
	Code         string
	WebhookURL   string
}

type JobResult struct {
	ID           string
	Status       string
	Result       ExecutionReport
	ErrorMessage string
}

package models

type ExecutionReport struct {
	Results []TestCaseResult `json:"results"`
}

type TestCaseResult struct {
	ID      string `json:"id"`
	Status  string `json:"status"` // AC, WA, TLE, RTE, IER
	TimeMS  int64  `json:"time_ms"`
	Message string `json:"message,omitempty"`
}

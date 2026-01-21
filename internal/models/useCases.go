package models

type ProblemUseCasesLimits struct {
	ProblemID string           `json:"problemaUUID"`
	UseCases  []UseCases       `json:"casosTeste"`
	Limits    []LanguageLimits `json:"limites"`
}

type UseCases struct {
	ExpectedInput  string `json:"entrada"`
	ExpectedOutput string `json:"saida"`
}

type LanguageLimits struct {
	Name             string `json:"language"`
	MaximumRamMB     int    `json:"memory_limit"`
	TimeLimitSeconds int    `json:"time_limit"`
}

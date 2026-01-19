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
	Name             string `json:"nome"`
	MaximumRamMB     int    `json:"ramMaximoEmMb"`
	TimeLimitSeconds int    `json:"tempoMaximoEmSegundos"`
}

package models

type ProblemUseCases struct {
	ProblemID int        `json:"problemaID"`
	UseCases  []UseCases `json:"casosTeste"`
}

type UseCases struct {
	ExpectedInput  string `json:"entrada"`
	ExpectedOutput string `json:"saida"`
}

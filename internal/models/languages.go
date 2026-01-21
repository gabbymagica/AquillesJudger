package models

type LanguageID int

const (
	Python LanguageID = 1
)

func (l LanguageID) String() string {
	switch l {
	case Python:
		return "Python"
	default:
		return "Unknown"
	}
}

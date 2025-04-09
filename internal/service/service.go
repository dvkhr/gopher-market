package service

import (
	"gopher-market/internal/store"
	"regexp"
)

type Service struct {
	Repo store.Database
}

func NewService(repo store.Database) *Service {
	return &Service{Repo: repo}
}

var numericRegex = regexp.MustCompile(`^[0-9]+$`)

func IsNumeric(word string) bool {
	return numericRegex.MatchString(word)
}

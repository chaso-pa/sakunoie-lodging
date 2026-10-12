package services

import (
	"errors"
	"strings"
)

var (
	ErrPostNotFound  = errors.New("post not found")
	ErrSpotNotFound  = errors.New("spot not found")
	ErrVisitNotFound = errors.New("visit not found")
	// ErrForbidden means the row exists but belongs to another member.
	ErrForbidden = errors.New("not the owner")
)

// ValidationError is a rejected input whose message is shown to the client.
type ValidationError struct {
	Message string
}

func (e *ValidationError) Error() string { return e.Message }

func invalid(message string) error {
	return &ValidationError{Message: message}
}

// escapeLike makes s match literally inside a LIKE pattern.
var likeEscaper = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)

func containsPattern(s string) string {
	return "%" + likeEscaper.Replace(s) + "%"
}

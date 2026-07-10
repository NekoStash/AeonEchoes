package repository

import (
	"errors"
	"fmt"
	"strings"
)

// ErrorKind identifies a persistence contract failure without coupling callers to error text.
type ErrorKind string

const (
	ErrorKindNotFound ErrorKind = "not_found"
	ErrorKindConflict ErrorKind = "conflict"
)

// Error reports a typed repository failure for HTTP and workflow callers.
type Error struct {
	Kind     ErrorKind
	Resource string
	ID       string
	Message  string
	Cause    error
}

func (e *Error) Error() string {
	if e == nil {
		return "repository error"
	}
	if message := strings.TrimSpace(e.Message); message != "" {
		return message
	}
	resource := strings.TrimSpace(e.Resource)
	if resource == "" {
		resource = "resource"
	}
	if id := strings.TrimSpace(e.ID); id != "" {
		return fmt.Sprintf("%s %q %s", resource, id, e.Kind)
	}
	return fmt.Sprintf("%s %s", resource, e.Kind)
}

func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Cause
}

func NotFound(resource, id string) error {
	resource = strings.TrimSpace(resource)
	id = strings.TrimSpace(id)
	message := fmt.Sprintf("%s %q not found", resource, id)
	return &Error{Kind: ErrorKindNotFound, Resource: resource, ID: id, Message: message}
}

func Conflict(resource, id, message string, cause error) error {
	return &Error{Kind: ErrorKindConflict, Resource: strings.TrimSpace(resource), ID: strings.TrimSpace(id), Message: strings.TrimSpace(message), Cause: cause}
}

func IsKind(err error, kind ErrorKind) bool {
	var repositoryError *Error
	return errors.As(err, &repositoryError) && repositoryError.Kind == kind
}

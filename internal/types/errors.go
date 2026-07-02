package types

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

var (
	ErrAlreadyExist = errors.New("already exist")
	ErrIsNotExist   = errors.New("is not exist")
	ErrValidation   = errors.New("invalid")
)

type AgentMistakeError struct {
	msg string
}

func NewAgentMistakeError(msg string) error {
	return &AgentMistakeError{
		msg: msg,
	}
}

func NewAgentMistakeErrorf(format string, arg ...any) error {
	return &AgentMistakeError{
		msg: fmt.Sprintf(format, arg...),
	}
}

func (e *AgentMistakeError) Error() string   { return "Agent make mistakes" }
func (e *AgentMistakeError) Message() string { return e.msg }

type Validator interface {
	Validate(context.Context) error
}

type ValidationError struct {
	problems map[string]string
}

func NewValidationError(problems map[string]string) error {
	return &ValidationError{problems: problems}
}

func (e *ValidationError) Error() string               { return "validation failed" }
func (e *ValidationError) Problems() map[string]string { return e.problems }
func (e *ValidationError) Message() string {

	var sb strings.Builder
	for k, v := range e.Problems() {
		fmt.Fprintf(&sb, "problem with %s - %s", k, v)
	}

	return sb.String()

}

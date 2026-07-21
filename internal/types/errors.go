package types

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
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
		fmt.Fprintf(&sb, "problem with %s - %s\n", k, v)
	}

	return sb.String()

}

func ResovleValidationProblems(err error) map[string]string {
	var validationErr *ValidationError
	if errors.As(err, &validationErr) {
		return validationErr.problems
	}
	return nil
}

func DistillErrNotExist(msg string, err error) error {
	errWrapper, ok := err.(interface{ Unwrap() []error })
	if !ok {
		if errors.Is(err, ErrIsNotExist) {
			slog.Warn(msg, "error", err)
			return nil
		}
		return err
	}

	var unExpectedErrs []error
	for _, werr := range errWrapper.Unwrap() {
		if errors.Is(werr, ErrIsNotExist) {
			slog.Warn(msg, "error", werr)
			continue
		}

		unExpectedErrs = append(unExpectedErrs, err)
	}
	return errors.Join(unExpectedErrs...)
}

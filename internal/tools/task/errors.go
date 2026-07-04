package tasktools

import (
	"arch-agent/internal/task"
	"arch-agent/internal/types"
	"errors"
)

func unwrapValidationError(err error) error {
	var ve *types.ValidationError
	if errors.As(err, &ve) {
		return types.NewAgentMistakeErrorf("invalid task \n%s", ve.Message())
	}
	return err
}

func mapSvcErrors(err error) error {
	switch {
	case
		errors.Is(err, task.ErrAlreadyExist) ||
			errors.Is(err, task.ErrIsNotExist) ||
			errors.Is(err, task.ErrCron) ||
			errors.Is(err, task.ErrNoRecipients):
		return types.NewAgentMistakeError(err.Error())
	default:
		return err
	}
}

package tasktools

import (
	"arch-agent/internal/task"
	"arch-agent/internal/types"
	"errors"
)

func unwrapValidationError(err error) error {
	var ve *types.ValidationError
	if errors.As(err, &ve) {
		return types.NewAgentMistakeErrorf("invalid task\n%s", ve.Message())
	}
	return err
}

func mapSvcErrors(err error) error {
	switch {
	case errors.Is(err, task.ErrAlreadyExist):
		return types.NewAgentMistakeError("task already exist")
	case errors.Is(err, task.ErrIsNotExist):
		return types.NewAgentMistakeError("task is not exist")
	case errors.Is(err, task.ErrTaskIsNotRunning):
		return types.NewAgentMistakeError("task already is not running")
	case errors.Is(err, task.ErrAlreadyRun):
		return types.NewAgentMistakeError("task already run")
	case errors.Is(err, task.ErrCron):
		return types.NewAgentMistakeError("bad cron format")
	default:
		return err
	}
}

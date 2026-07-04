package task

import (
	"arch-agent/internal/agent"
	"arch-agent/internal/types"
	"errors"
	"regexp"
	"time"
)

var ErrIsNotExist = types.ErrIsNotExist
var ErrAlreadyExist = types.ErrAlreadyExist
var ErrCron = errors.New("cron is not support this expression")
var ErrNoRecipients = errors.New("must contain at least one recipient")

type ValidationError types.ValidationError

type Cron interface {
	NextTime() time.Duration
	Expression() string
}

var cronRegex = regexp.MustCompile(`^(((?:[1-5]?[0-9])|(?:\*))(?:\/\d+)?(?:,(?:(?:[1-5]?[0-9])|(?:\*))(?:\/\d+)?)*)\s+(((?:[0-1]?[0-9]|2[0-3])|(?:\*))(?:\/\d+)?(?:,(?:(?:[0-1]?[0-9]|2[0-3])|(?:\*))(?:\/\d+)?)*)\s+(((?:[0-2]?[0-9]|3[0-1])|(?:\*))(?:\/\d+)?(?:,(?:(?:[0-2]?[0-9]|3[0-1])|(?:\*))(?:\/\d+)?)*)\s+(((?:[0-9]|1[0-2])|(?:\*))(?:\/\d+)?(?:,(?:(?:[0-9]|1[0-2])|(?:\*))(?:\/\d+)?)*)\s+(((?:[0-6])|(?:\*))(?:\/\d+)?(?:,(?:(?:[0-6])|(?:\*))(?:\/\d+)?)*)$`)

type TaskRepo interface {
	All() (map[string]TaskConfig, error)
	Get(id string) (TaskConfig, error)
	Delete(id string) error
	Save(t TaskConfig) error
}

type TaskConfig struct {
	Name        string     `json:"name"`
	Description string     `json:"description"`
	Recipients  []agent.ID `json:"recipients"`
	Reglament   string     `json:"schedule"`
	Request     string     `json:"request"`
	Active      bool       `json:"active"`
	Oneshot     bool       `json:"oneshot"`
}

type TaskPatch struct {
	Active      *bool       `json:"active,omitempty"`
	Name        *string     `json:"name,omitempty"`
	Description *string     `json:"description,omitempty"`
	Recipients  *[]agent.ID `json:"recipients,omitempty"`
	Reglament   *string     `json:"schedule,omitempty"`
	Request     *string     `json:"request,omitempty"`
	Oneshot     *bool       `json:"oneshot,omitempty"`
}

func applyPatch(cfg TaskConfig, patch TaskPatch) TaskConfig {
	if patch.Active != nil {
		cfg.Active = *patch.Active
	}

	if patch.Name != nil {
		cfg.Name = *patch.Name
	}

	if patch.Description != nil {
		cfg.Description = *patch.Description
	}

	if patch.Recipients != nil {
		cfg.Recipients = append([]agent.ID{}, (*patch.Recipients)...)
	}

	if patch.Reglament != nil {
		cfg.Reglament = *patch.Reglament
	}

	if patch.Request != nil {
		cfg.Request = *patch.Request
	}

	if patch.Oneshot != nil {
		cfg.Oneshot = *patch.Oneshot
	}

	return cfg
}

func validateTaskConfig(cfg TaskConfig) error {
	problems := make(map[string]string)
	if cfg.Name == "" {
		problems["name"] = "must be not empty"
	}
	if cfg.Description == "" {
		problems["description"] = "must be not empty"
	}
	if !(len(cfg.Recipients) > 0) {
		problems["recipients"] = ErrNoRecipients.Error()
	}
	if !cronRegex.MatchString(cfg.Reglament) {
		problems["reglament"] = "invalid format"
	}
	if cfg.Request == "" {
		problems["request"] = "must be not empty"
	}
	if len(problems) > 0 {
		return types.NewValidationError(problems)
	}

	return nil
}

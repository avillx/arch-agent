package task

import (
	"arch-agent/internal/agent"
	"arch-agent/internal/types"
	"regexp"
	"time"
)

type ValidationError types.ValidationError

type Cron interface {
	NextTime() time.Duration
	Expression() string
}

var cronRegex = regexp.MustCompile(`^(((?:[1-5]?[0-9])|(?:\*))(?:\/\d+)?(?:,(?:(?:[1-5]?[0-9])|(?:\*))(?:\/\d+)?)*)\s+(((?:[0-1]?[0-9]|2[0-3])|(?:\*))(?:\/\d+)?(?:,(?:(?:[0-1]?[0-9]|2[0-3])|(?:\*))(?:\/\d+)?)*)\s+(((?:[0-2]?[0-9]|3[0-1])|(?:\*))(?:\/\d+)?(?:,(?:(?:[0-2]?[0-9]|3[0-1])|(?:\*))(?:\/\d+)?)*)\s+(((?:[0-9]|1[0-2])|(?:\*))(?:\/\d+)?(?:,(?:(?:[0-9]|1[0-2])|(?:\*))(?:\/\d+)?)*)\s+(((?:[0-6])|(?:\*))(?:\/\d+)?(?:,(?:(?:[0-6])|(?:\*))(?:\/\d+)?)*)$`)

type TaskRecord struct {
	Active bool `json:"active"`
	*TaskConfig
}
type TaskConfig struct {
	name        string
	description string
	recipients  []agent.ID
	reglament   string
	request     string
	oneshot     bool
}

func (c *TaskConfig) Name() string           { return c.name }
func (c *TaskConfig) Description() string    { return c.description }
func (c *TaskConfig) Recipients() []agent.ID { return c.recipients }
func (c *TaskConfig) Reglament() string      { return c.reglament }
func (c *TaskConfig) Request() string        { return c.request }
func (c *TaskConfig) Oneshot() bool          { return c.oneshot }

func NewValidTaskConfig(
	name string,
	description string,
	recipients []agent.ID,
	reglament string,
	request string,
	oneshot bool,
) (*TaskConfig, error) {

	cfg := &TaskConfig{
		name:        name,
		description: description,
		recipients:  recipients,
		reglament:   reglament,
		request:     request,
		oneshot:     oneshot,
	}

	if err := validateTaskConfig(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

func validateTaskConfig(cfg *TaskConfig) error {
	problems := make(map[string]string)
	if cfg.name == "" {
		problems["name"] = "must be not empty"
	}
	if cfg.description == "" {
		problems["description"] = "must be not empty"
	}
	if !(len(cfg.recipients) > 0) {
		problems["recipients"] = "must contain at least one recipient"
	}
	if !cronRegex.MatchString(cfg.reglament) {
		problems["reglament"] = "invalid format"
	}
	if cfg.request == "" {
		problems["request"] = "must be not empty"
	}
	if len(problems) > 0 {
		return types.NewValidationError(problems)
	}

	return nil
}

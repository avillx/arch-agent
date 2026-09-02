package hooks

import (
	"arch-agent/internal/agent"
	"arch-agent/internal/runtime"
	"context"
)

type Replcaer interface {
	Replace(text string) string
}

var _ runtime.CompletionHook = (*NeverTypeSecretsHook)(nil)

// *agent.Completion - secrets -> placeholders (agent type secret by itself)
type NeverTypeSecretsHook struct {
	r Replcaer
}

func (h *NeverTypeSecretsHook) Apply(
	ctx context.Context,
	c *agent.Completion,
) (*agent.Completion, error) {

	c.Content = h.r.Replace(c.Content)
	return c, nil
}

var _ runtime.ToolCallHook = (*NeverToolCallSecretsHook)(nil)

// *agent.ToolCall   - secrets -> placeholders (agent type secret in tool)
type NeverToolCallSecretsHook struct {
	r Replcaer
}

func (h *NeverToolCallSecretsHook) Apply(
	ctx context.Context,
	c *agent.ToolCall,
) (*agent.ToolCall, error) {

	stringArgs := string(c.Arguments)
	safeString := h.r.Replace(stringArgs)
	c.Arguments = []byte(safeString)

	return c, nil
}

var _ runtime.ToolResultHook = (*NeverReadSecretsHook)(nil)

// *AfterToolCall    - secrets -> placeholder (agent read the secrets)
type NeverReadSecretsHook struct {
	r Replcaer
}

func (h *NeverReadSecretsHook) Apply(
	ctx context.Context,
	c *runtime.AfterToolCall,
) (*runtime.AfterToolCall, error) {

	safeCps := []agent.ContentPart{}
	for _, cp := range c.Result {
		safe := h.r.Replace(cp.Text)
		safeCps = append(safeCps, agent.ContentPart{Text: safe, ImageURL: cp.ImageURL})
	}

	c.Result = safeCps

	return c, nil
}

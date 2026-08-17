package hooks

import (
	"arch-agent/internal/agent"
	"arch-agent/internal/runtime"
	"arch-agent/internal/types"
	"context"
)

var _ runtime.ToolResultHook = (*ContentSizeLimitHook)(nil)

type ContentSizeLimitHook struct {
	limitBytes int
}

func (h *ContentSizeLimitHook) Apply(
	ctx context.Context,
	tc *runtime.AfterToolCall,
) (*runtime.AfterToolCall, error) {

	// validation
	var buf []byte
	for _, p := range tc.Result {
		buf = append(buf, []byte(p.Text)...)
	}

	if len(buf) <= h.limitBytes {
		return tc, nil
	}

	// truncation
	truncatedContent := agent.NewContent(string(buf[:h.limitBytes]))

	var imagesContent []agent.ContentPart
	for _, p := range tc.Result {
		if p.ImageURL != "" {
			imagesContent = append(imagesContent, agent.ContentPart{ImageURL: p.ImageURL})
		}
	}
	tc.Result = append(truncatedContent, imagesContent...)

	// representation
	keptPercent := h.limitBytes * 100 / len(buf)
	err := types.NewAgentMistakeErrorf(
		"result is over %d kb, and was truncated to %d%%",
		h.limitBytes/1024, // repr in kilobites
		keptPercent,
	)

	return tc, err
}

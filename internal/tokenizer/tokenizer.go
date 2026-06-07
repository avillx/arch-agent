package tiktoken

import (
	"arch-agent/internal/agent"
	"log/slog"
	"strings"

	"github.com/tiktoken-go/tokenizer"
	"github.com/tiktoken-go/tokenizer/codec"
)

var _ agent.TokenCounter = (*TikToken)(nil)

type TikToken struct {
	tokenizer tokenizer.Codec
}

func New() *TikToken {
	return &TikToken{
		tokenizer: codec.NewCl100kBase(),
	}
}

func (c *TikToken) RawString(content string) int {
	ids, _, err := c.tokenizer.Encode(content)

	if err != nil {
		slog.Error("bad tokenization")
	}
	return len(ids)
}

func (c *TikToken) Tools(tools []agent.Tool) int {

	var heap strings.Builder
	for _, t := range tools {
		heap.WriteString(string(t.Name()))
		heap.WriteString(t.Description())
		for _, p := range t.Schema() {

			heap.WriteString(p.Name)
			heap.WriteString(p.Description)
			heap.WriteString(string(p.Type))

			if len(p.Enum) > 0 {
				heap.WriteString(strings.Join(p.Enum, " "))
			}

		}
	}

	ids, _, _ := c.tokenizer.Encode(heap.String())
	return len(ids)
}

func (c *TikToken) Messages(msgs []agent.Message) int {

	var heap strings.Builder
	for _, msg := range msgs {
		heap.WriteString(msg.Content())

		switch typedMsg := msg.(type) {
		case agent.AgentMessage:
			for _, call := range typedMsg.ToolCalls() {
				heap.WriteString(call.ID)
				heap.WriteString(string(call.ToolName))
				heap.Write(call.Arguments)
			}
		case agent.ToolResultMessage:
			heap.WriteString(typedMsg.ToolCallID())
		}
	}

	ids, _, _ := c.tokenizer.Encode(heap.String())
	return len(ids)
}

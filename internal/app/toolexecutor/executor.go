package tools

import (
	"arch-agent/internal/domain/conversation"
	"context"
	"sync"

	"golang.org/x/sync/errgroup"
)

type ToolCallRecivier interface {
	SendCall(ctx context.Context, toolName string, args conversation.ToolArguments) (string, error)
	Tools() []ToolDefinition
}

type Executor struct {
	recivier ToolCallRecivier
}

func NewExecutor(r ToolCallRecivier) *Executor {
	return &Executor{
		recivier: r,
	}
}

func (e *Executor) Execute(ctx context.Context, calls []conversation.ToolCall) ([]conversation.ToolCallResult, error) {
	var mu sync.Mutex
	g, ctx := errgroup.WithContext(ctx)
	results := []conversation.ToolCallResult{}

	for _, c := range calls {
		c := c // shadowing for avoid implicit racing

		g.Go(func() error {
			result, err := e.recivier.SendCall(ctx, c.ToolName(), c.Arguments())
			if err != nil {
				return err
			}

			// for exclude racing for results slice
			mu.Lock()
			defer mu.Unlock()

			results = append(results, conversation.NewToolCallResult(c.ID(), result))
			return nil
		})

	}

	return results, g.Wait()
}


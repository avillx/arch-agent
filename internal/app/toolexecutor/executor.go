package toolexecutor

import (
	ports "arch-agent/internal/app"
	"arch-agent/internal/domain/shared"
	"context"
	"sync"

	"golang.org/x/sync/errgroup"
)

type ToolExecutor struct {
	ports.Recivier
}

func (e *ToolExecutor) ExecuteToolCalls(ctx context.Context, calls []shared.ToolCall) ([]*shared.ToolResultMessage, error) {
	var mu sync.Mutex
	g, ctx := errgroup.WithContext(ctx)
	results := []*shared.ToolResultMessage{}

	for _, c := range calls {
		g.Go(func() error {
			result, err := e.Execute(ctx, c.ToolName, c.Arguments)
			if err != nil {
				return err
			}

			// for exclude racing for results slice
			mu.Lock()
			defer mu.Unlock()

			results = append(results, shared.NewToolResultMessage(c.ID, result))
			return nil
		})

	}

	return results, g.Wait()
}

// func (e *ToolExecutor) executeToolCall(ctx context.Context, call shared.ToolCall) (string, error) {

// 	if itool, ok := e.internalTools[call.ToolName]; ok {
// 		return itool.Execute(ctx, call.Arguments)
// 	}
// 	return "", nil
// }

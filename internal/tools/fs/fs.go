package fstools

import (
	"arch-agent/internal/agent"
	"arch-agent/internal/files"
	rf "arch-agent/internal/files/rule"
	ruledfiles "arch-agent/internal/files/rule"
	"arch-agent/internal/runtime"
	"arch-agent/internal/types"
	"context"
	"errors"
	"fmt"
	"strings"
)

type RuledAccessFactory func(ctx context.Context) (*ruledfiles.RuledFileSystem, error)

func NewAgentAccessRuledFS(fs *files.FileSystem, repo agent.Repo) RuledAccessFactory {
	return func(ctx context.Context) (*ruledfiles.RuledFileSystem, error) {
		agentID, ok := runtime.AgentIDFromContext(ctx)
		if !ok {
			return nil, fmt.Errorf("no agent in context")
		}
		agt, err := repo.Get(agentID)
		if err != nil {
			return nil, err
		}
		return rf.NewRuledFileSystem(fs, rf.AgentAccessRules(agt)...)
	}
}

func NewMemoryAccessRuledFS(fs *files.FileSystem) RuledAccessFactory {
	return func(ctx context.Context) (*ruledfiles.RuledFileSystem, error) {
		agentID, ok := runtime.AgentIDFromContext(ctx)
		if !ok {
			return nil, fmt.Errorf("no agent in context")
		}
		return rf.NewRuledFileSystem(fs, rf.AgentMemoryAccessRules(agentID)...)
	}
}

func matchLines(agentPath, content, query string, limit int) []string {
	lower := strings.ToLower(query)
	var matches []string
	for i, line := range strings.Split(content, "\n") {
		if len(matches) >= limit {
			break
		}
		if strings.Contains(strings.ToLower(line), lower) {
			matches = append(matches, fmt.Sprintf("%s:%d: %s", agentPath, i+1, strings.TrimSpace(line)))
		}
	}
	return matches
}

func ruleBreakToAgentMistake(err error) error {
	var ruleError *rf.RuleError
	if errors.As(err, &ruleError) {
		return types.NewAgentMistakeError(err.Error())
	}
	return err
}

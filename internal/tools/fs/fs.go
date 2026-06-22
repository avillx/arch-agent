package fstools

import (
	"arch-agent/internal/agent"
	"arch-agent/internal/files"
	rf "arch-agent/internal/files/rule"
	"arch-agent/internal/runtime"
	"context"
	"fmt"
	"strings"
)

func newRuledFS(ctx context.Context, fs *files.FileSystem, repo agent.Repo) (*rf.RuledFileSystem, error) {
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
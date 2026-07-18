package api

import (
	"arch-agent/internal/agent"
	"arch-agent/internal/memory"
	"arch-agent/internal/runtime"
	"arch-agent/internal/session"
	"arch-agent/internal/types"
	"errors"
	"net/http"
	"path"
	"strings"
)

type memoryHandler struct {
	memorySvc     *memory.Memory
	memoryIndexer agent.MemoryIndexer
	memoryRepo    agent.MemoryRepo
}

// GET /memory/{agent}
func (h *memoryHandler) List(w http.ResponseWriter, r *http.Request) error {

	agentID := r.PathValue("agent")

	idx, err := h.memoryIndexer.MemoryIndex(agent.ID(agentID))
	if err != nil {
		return internal(err)
	}

	memories := map[string]string{}
	for k, v := range idx {
		memories[strings.TrimSuffix(path.Base(k), path.Ext(k))] = v
	}

	return respond(w, http.StatusOK, memories)
}

// GET /memory/{agent}/{memory_name}
func (h *memoryHandler) Get(w http.ResponseWriter, r *http.Request) error {
	agentID := r.PathValue("agent")
	memoryName := r.PathValue("memory_name")

	content, err := h.memoryRepo.GetMemory(agent.ID(agentID), memoryName)
	if err != nil {
		return internal(err)
	}

	return respond(w, http.StatusOK, map[string]any{
		"agent":       agentID,
		"memory_name": memoryName,
		"content":     content,
	})
}

// POST /memory/{agent}/consolidate
func (h *memoryHandler) Consolidate(w http.ResponseWriter, r *http.Request) error {
	agentID := r.PathValue("agent")

	evCh := make(chan runtime.Event, 16)

	stream := newStream(w)
	defer stream.done()

	evReader := runtime.EventReader{
		OnComplete: func(_ agent.ID, _ session.ID, c *agent.Completion) {
			stream.send(CompletionDTO{
				Done:      c.Done,
				Content:   c.Content,
				ToolCalls: toolCallsToDTO(c.ToolCalls),
			})
		},
	}
	go evReader.Read(evCh)

	if err := h.memorySvc.DreamImmidate(r.Context(), agent.ID(agentID), evCh); err != nil {
		if errors.Is(err, types.ErrIsNotExist) {
			return badRequest("agent is not exist")
		}
		return internal(err)
	}

	return nil
}

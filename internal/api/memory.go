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

	type MemoryRecordDTO struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}

	type MemoryIndexDTO struct {
		Agent   agent.ID          `json:"agent"`
		Records []MemoryRecordDTO `json:"memory_records"`
	}

	agentID := agent.ID(r.PathValue("agent"))

	idx, err := h.memoryIndexer.MemoryIndex(agentID)
	if err != nil {
		return internal(err)
	}

	memories := []MemoryRecordDTO{}
	for k, v := range idx {
		memories = append(memories, MemoryRecordDTO{
			Name:        strings.TrimSuffix(path.Base(k), path.Ext(k)),
			Description: v,
		})
	}

	dto := MemoryIndexDTO{
		Agent:   agentID,
		Records: memories,
	}

	return respond(w, http.StatusOK, dto)
}

// GET /memory/{agent}/{memory_name}
func (h *memoryHandler) Get(w http.ResponseWriter, r *http.Request) error {

	type MemoryDTO struct {
		Agent   agent.ID `json:"agent"`
		Name    string   `json:"memory_name"`
		Content string   `json:"content"`
	}

	agentID := agent.ID(r.PathValue("agent"))
	memoryName := r.PathValue("memory_name")

	content, err := h.memoryRepo.GetMemory(agentID, memoryName)
	if err != nil {
		return internal(err)
	}

	dto := MemoryDTO{
		Agent:   agentID,
		Name:    memoryName,
		Content: content,
	}

	return respond(w, http.StatusOK, dto)
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

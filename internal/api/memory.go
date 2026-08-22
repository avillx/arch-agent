package api

import (
	"arch-agent/internal/agent"
	"arch-agent/internal/memory"
	"arch-agent/internal/runtime"
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

func NewMemoryHandler(
	memorySvc *memory.Memory,
	memoryIndexer agent.MemoryIndexer,
	memoryRepo agent.MemoryRepo,
) *memoryHandler {
	return &memoryHandler{
		memorySvc:     memorySvc,
		memoryIndexer: memoryIndexer,
		memoryRepo:    memoryRepo,
	}
}

// GET /memory/{agent}
func (h *memoryHandler) List(w http.ResponseWriter, r *http.Request) Response {

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
		return NewInternalError(err)
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

	return NewJSONResponse(http.StatusOK, dto)
}

// GET /memory/{agent}/{memory_name}
func (h *memoryHandler) Get(w http.ResponseWriter, r *http.Request) Response {

	type MemoryDTO struct {
		Agent   agent.ID `json:"agent"`
		Name    string   `json:"memory_name"`
		Content string   `json:"content"`
	}

	agentID := agent.ID(r.PathValue("agent"))
	memoryName := r.PathValue("memory_name")

	content, err := h.memoryRepo.GetMemory(agentID, memoryName)
	if err != nil {

		// TODO: agent not found

		// TODO: memory not found

		return NewInternalError(err)
	}

	dto := MemoryDTO{
		Agent:   agentID,
		Name:    memoryName,
		Content: content,
	}

	return NewJSONResponse(http.StatusOK, dto)
}

// POST /memory/{agent}/consolidate
func (h *memoryHandler) Consolidate(w http.ResponseWriter, r *http.Request) Response {
	agentID := agent.ID(r.PathValue("agent"))

	stream := newStream(w)
	defer stream.close()

	evCh := make(chan runtime.Event, 16)
	defer close(evCh)

	// stream completion events
	go func() {
		for ev := range evCh {
			if completeEvent, ok := ev.(runtime.CompleteEvent); ok {
				c := completeEvent.Complete()
				stream.send(CompletionDTO{
					Done:      c.Done,
					Content:   c.Content,
					ToolCalls: toolCallsToDTO(c.ToolCalls),
				})
			}
		}
	}()

	if err := h.memorySvc.ConsolidateImmidate(r.Context(), agentID, evCh); err != nil {
		if errors.Is(err, types.ErrIsNotExist) {
			return NewNotFound("agent is not exist")
		}
		return NewInternalError(err)
	}

	return nil
}

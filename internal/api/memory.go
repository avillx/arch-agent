package api

import (
	"arch-agent/internal/agent"
	"arch-agent/internal/memory"
	"arch-agent/internal/runtime"
	"arch-agent/internal/types"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
)

type memoryHandler struct {
	consolidationSvc *memory.ConsolidationService
	memoryIndexer    agent.MemoryIndexer
	memoryRepo       agent.MemoryRepo
	logger           *slog.Logger
}

func NewMemoryHandler(
	consolidationSvc *memory.ConsolidationService,
	memoryIndexer agent.MemoryIndexer,
	memoryRepo agent.MemoryRepo,
	logger *slog.Logger,
) *memoryHandler {
	return &memoryHandler{
		consolidationSvc: consolidationSvc,
		memoryIndexer:    memoryIndexer,
		memoryRepo:       memoryRepo,
		logger:           logger.WithGroup("memory"),
	}
}

// GET /memory/{agent}
func (h *memoryHandler) List(w http.ResponseWriter, r *http.Request) Response {

	type MemoryRecordDTO map[string]string

	agentID := agent.ID(r.PathValue("agent"))

	idx, err := h.memoryIndexer.MemoryIndex(agentID)
	if err != nil {
		if errors.Is(err, types.ErrIsNotExist) {
			return NewNotFound("agent memory is not found")
		}
		return NewInternalError(err)
	}

	return NewJSONResponse(http.StatusOK, MemoryRecordDTO(idx))
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
		if errors.Is(err, types.ErrIsNotExist) {
			return NewBadRequest("this memory is not exist")
		}
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

	if err := h.consolidationSvc.ConsolidateImmidate(r.Context(), agentID, evCh); err != nil {
		if errors.Is(err, types.ErrIsNotExist) {
			stream.sendError(http.StatusNotFound, err)
			return nil
		}

		if errors.Is(err, memory.ErrDisabledMemory) {
			stream.sendError(http.StatusBadRequest, err)
			return nil
		}

		h.logger.Error("consolidation internal error occured", "error", err)
		stream.sendError(http.StatusInternalServerError, fmt.Errorf("internal error occured"))
		return nil
	}

	return nil
}

// GET /memory/config
func (h *memoryHandler) GetConfig(w http.ResponseWriter, r *http.Request) Response {
	cfg := h.consolidationSvc.Config()
	return NewJSONResponse(http.StatusOK, cfg)
}

// POST /memory/config
func (h *memoryHandler) SetConfig(w http.ResponseWriter, r *http.Request) Response {

	newCfg, err := decode[memory.ConsolidatorConfig](r)
	if err != nil {
		return NewInvalidRequest(err)
	}

	if err := h.consolidationSvc.SaveConfig(newCfg); err != nil {
		if errors.Is(err, types.ErrIsNotExist) {
			return NewNotFound("model is not exist")
		}
		return NewInternalError(err)
	}

	return NewResponse(http.StatusOK)
}

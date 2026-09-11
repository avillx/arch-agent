package api

import (
	"arch-agent/internal/agent"
	"arch-agent/internal/types"
	"context"
	"errors"
	"net/http"
)

type AgentDTO struct {
	Model        string   `json:"model,omitempty"`
	Memory       bool     `json:"memory,omitempty"`
	Description  string   `json:"description,omitempty"`
	ToolServers  []string `json:"tool_servers,omitempty"`
	SystemPrompt string   `json:"system_prompt,omitempty"`
}

func (d AgentDTO) Validate(_ context.Context) error {
	if d.Model == "" {
		return types.NewValidationError(map[string]string{
			"model": "must specified",
		})
	}
	return nil
}

type agentHandler struct {
	repo agent.Repo
}

// GET /agent
func (h *agentHandler) List(w http.ResponseWriter, r *http.Request) Response {

	agents, err := h.repo.All()
	if err != nil {
		return NewInternalError(err)
	}

	dtos := []AgentDTO{}
	for _, agt := range agents {
		dtos = append(dtos, agentToDTO(agt))
	}

	return NewJSONResponse(http.StatusOK, dtos)
}

// POST /agent/{id} DTO
func (h *agentHandler) Create(w http.ResponseWriter, r *http.Request) Response {
	id := agent.ID(r.PathValue("id"))

	_, err := h.repo.Get(id)
	if err == nil {
		return NewBadRequest("already exist")
	}
	if !errors.Is(err, types.ErrIsNotExist) {
		return NewInternalError(err)
	}

	return h.saveAgt(id, r)
}

// PUT /agent/{id}
func (h *agentHandler) Update(w http.ResponseWriter, r *http.Request) Response {
	id := agent.ID(r.PathValue("id"))

	// To ensure existence
	_, err := h.repo.Get(id)
	if err != nil {
		if errors.Is(err, types.ErrIsNotExist) {
			return NewBadRequest("agent is not exist")
		}
		return NewInternalError(err)
	}

	return h.saveAgt(id, r)
}

func (h *agentHandler) saveAgt(agentID agent.ID, r *http.Request) Response {
	agentDTO, err := decode[AgentDTO](r)
	if err != nil {
		return NewInvalidRequest(err)
	}

	if err := h.repo.Save(dtoToAgent(agentID, agentDTO)); err != nil {

		// On Model not exist or ToolServer not exist
		if errors.Is(err, types.ErrIsNotExist) {
			return NewBadRequest(err.Error())
		}

		return NewInternalError(err)
	}

	return NewResponse(http.StatusOK)
}

// GET /agent/{id} // DTO
func (h *agentHandler) Read(w http.ResponseWriter, r *http.Request) Response {

	id := agent.ID(r.PathValue("id"))
	agt, err := h.repo.Get(id)
	if err != nil {
		if errors.Is(err, types.ErrIsNotExist) {
			return NewBadRequest("agent is not exist")
		}
		return NewInternalError(err)
	}

	return NewJSONResponse(http.StatusOK, agentToDTO(agt))
}

// DELETE /agent/{id}
func (h *agentHandler) Delete(w http.ResponseWriter, r *http.Request) Response {
	id := agent.ID(r.PathValue("id"))

	if err := h.repo.Delete(id); err != nil {
		if errors.Is(err, types.ErrIsNotExist) {
			return NewBadRequest(err.Error())
		}
		return NewInternalError(err)
	}

	return NewResponse(http.StatusOK)
}

func agentToDTO(agt agent.Agent) AgentDTO {
	return AgentDTO{
		Model:        string(agt.Model()),
		Memory:       agt.HasMemory(),
		Description:  agt.Description(),
		ToolServers:  agt.ToolServers(),
		SystemPrompt: agt.SystemPrompt(),
	}
}

func dtoToAgent(id agent.ID, dto AgentDTO) agent.Agent {
	return agent.NewAgent(
		id,
		dto.Description,
		dto.SystemPrompt,
		dto.Model,
		nil,
		dto.ToolServers,
		dto.Memory,
	)
}

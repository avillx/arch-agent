package api

import (
	"arch-agent/internal/agent"
	"arch-agent/internal/types"
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

type agentHandler struct {
	repo agent.Repo
}

// GET /agent
func (h *agentHandler) List(w http.ResponseWriter, r *http.Request) Response {

	type AgentListDTO struct {
		Agents []AgentDTO `json:"agents"`
	}

	agents, err := h.repo.All()
	if err != nil {
		return NewInternalError(err)
	}

	dtos := []AgentDTO{}
	for _, agt := range agents {
		dtos = append(dtos, agentToDTO(agt))
	}

	dto := AgentListDTO{Agents: dtos}

	return NewJSONResponse(http.StatusOK, dto)
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

	agentDTO, err := decode[AgentDTO](r)
	if err != nil {
		return NewInternalError(err)
	}

	if err := h.repo.Save(dtoToAgent(id, agentDTO)); err != nil {
		return NewInternalError(err)
	}

	return NewResponse(http.StatusOK)
}

// PUT /agent/{id}
func (h *agentHandler) Update(w http.ResponseWriter, r *http.Request) Response {
	id := agent.ID(r.PathValue("id"))

	_, err := h.repo.Get(id)
	if err != nil {
		if errors.Is(err, types.ErrIsNotExist) {
			return NewBadRequest("agent is not exist")
		}
		return NewInternalError(err)
	}

	updated, err := decode[AgentDTO](r)
	if err != nil {
		return NewInvalidRequest(err)
	}

	if err := h.repo.Save(dtoToAgent(id, updated)); err != nil {
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
	_, err := h.repo.Get(id)
	if err != nil {
		if errors.Is(err, types.ErrIsNotExist) {
			return NewBadRequest("agent is not exist")
		}
		return NewInternalError(err)
	}

	if err := h.repo.Delete(id); err != nil {
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

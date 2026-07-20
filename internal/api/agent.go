package api

import (
	"arch-agent/internal/agent"
	"arch-agent/internal/types"
	"errors"
	"fmt"
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

// get /agent/list
func (h *agentHandler) List(w http.ResponseWriter, r *http.Request) error {

	type AgentListDTO struct {
		Agents []AgentDTO `json:"agents"`
	}

	agents, err := h.repo.All()
	if err != nil {
		return internal(err)
	}

	dtos := []AgentDTO{}
	for _, agt := range agents {
		dtos = append(dtos, agentToDTO(agt))
	}

	return respond(w, http.StatusOK, AgentListDTO{Agents: dtos})
}

// POST /agent/{id} DTO
func (h *agentHandler) Create(w http.ResponseWriter, r *http.Request) error {
	id := agent.ID(r.PathValue("id"))

	_, err := h.repo.Get(id)
	if err == nil {
		return badRequest(fmt.Sprintf("agent %s already exist", id))
	}
	if !errors.Is(err, types.ErrIsNotExist) {
		return internal(err)
	}

	agentDTO, err := decode[AgentDTO](r)
	if err != nil {
		return badRequest("invalid json")
	}

	if err := h.repo.Save(dtoToAgent(id, agentDTO)); err != nil {
		return internal(err)
	}

	return respond(w, http.StatusCreated, message("agent created"))
}

// PUT /agent/{id}
func (h *agentHandler) Update(w http.ResponseWriter, r *http.Request) error {
	id := agent.ID(r.PathValue("id"))

	_, err := h.repo.Get(id)
	if err != nil {
		if errors.Is(err, types.ErrIsNotExist) {
			return badRequest("agent is not exist")
		}
		return internal(err)
	}

	updated, err := decode[AgentDTO](r)
	if err != nil {
		return badRequest("invalid json")
	}

	if err := h.repo.Save(dtoToAgent(id, updated)); err != nil {
		return internal(err)
	}

	return respond(w, http.StatusOK, message("agent updated"))
}

// GET /agent/{id} // DTO
func (h *agentHandler) Read(w http.ResponseWriter, r *http.Request) error {

	id := agent.ID(r.PathValue("id"))
	agent, err := h.repo.Get(id)
	if err != nil {
		if errors.Is(err, types.ErrIsNotExist) {
			return badRequest("agent is not exist")
		}
		return internal(err)
	}

	return respond(w, http.StatusOK, agentToDTO(agent))
}

// DELETE /agent/{id}
func (h *agentHandler) Delete(w http.ResponseWriter, r *http.Request) error {
	id := agent.ID(r.PathValue("id"))
	_, err := h.repo.Get(id)
	if err != nil {
		if errors.Is(err, types.ErrIsNotExist) {
			return badRequest("agent is not exist")
		}
		return internal(err)
	}

	if err := h.repo.Delete(id); err != nil {
		return internal(err)
	}

	return respond(w, http.StatusOK, message("agent deleted"))
}

func dtoToAgent(id agent.ID, dto AgentDTO) agent.Agent {
	return agent.NewAgent(
		id,
		dto.Description,
		dto.SystemPrompt,
		agent.ModelName(dto.Model),
		nil,
		dto.ToolServers,
		dto.Memory,
	)
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

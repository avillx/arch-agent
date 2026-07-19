package api

import (
	"arch-agent/internal/mcp"
	"arch-agent/internal/types"
	"errors"
	"net/http"
)

// handler
type mcpHandler struct {
	mcpSvc *mcp.Service
}

// GET /api/v1/mcp/list
func (h *mcpHandler) List(w http.ResponseWriter, _ *http.Request) error {

	type MCPToolServerReprDTO struct {
		Transport string `json:"transport"`
		ToolServerReprDTO
	}

	type MCPListResponseDTO struct {
		Servers []MCPToolServerReprDTO `json:"mcp_servers"`
	}

	toolServers := []MCPToolServerReprDTO{}
	for _, mcpServers := range h.mcpSvc.List() {

		toolList := []ToolReprDTO{}
		for _, t := range mcpServers.Tools() {
			toolList = append(toolList, ToolReprDTO{
				Name:        string(t.Name()),
				Description: t.Description(),
			})
		}

		serverRepr := MCPToolServerReprDTO{
			Transport: mcpServers.Gateway().Type(),
			ToolServerReprDTO: ToolServerReprDTO{
				Name:  string(mcpServers.ID()),
				Tools: toolList,
			},
		}

		toolServers = append(toolServers, serverRepr)
	}

	dto := MCPListResponseDTO{
		Servers: toolServers,
	}

	return respond(w, http.StatusOK, dto)
}

// POST /api/v1/mcp/connect
func (h *mcpHandler) Connect(w http.ResponseWriter, r *http.Request) error {

	gatewayConfig, err := decode[mcp.ServerGatewayConfig](r)
	if err != nil {
		return respond(w, http.StatusBadRequest, "invalid json")
	}

	id, err := h.mcpSvc.Connect(r.Context(), gatewayConfig)
	if err != nil {
		var validationErr *types.ValidationError
		if errors.As(err, &validationErr) {
			return invalidRequest(validationErr.Problems())
		}
		if errors.Is(err, types.ErrAlreadyExist) {
			return badRequest("already exist")
		}

		return internal(err)
	}

	return respond(w, http.StatusOK, map[string]string{
		"msg":        "success",
		"created_id": string(id),
	})
}

// GET /api/v1/mcp/disconnect/{id}
func (h *mcpHandler) Disconnect(w http.ResponseWriter, r *http.Request) error {
	mcpID := r.PathValue("id")

	if err := h.mcpSvc.Disconnect(mcp.MCPServerID(mcpID)); err != nil {
		if errors.Is(err, types.ErrIsNotExist) {
			return badRequest(err.Error())
		}
		return internal(err)
	}
	return respond(w, http.StatusOK, message("disconnected"))
}

// POST /api/v1/reload
func (h *mcpHandler) Reload(w http.ResponseWriter, r *http.Request) error {
	if err := h.mcpSvc.Reload(r.Context()); err != nil {
		return internal(err)
	}
	return respond(w, http.StatusOK, message("reloaded"))
}

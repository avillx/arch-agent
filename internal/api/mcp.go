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

// GET /mcp
func (h *mcpHandler) List(w http.ResponseWriter, _ *http.Request) Response {

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

	return NewJSONResponse(http.StatusOK, dto)
}

// POST /mcp
func (h *mcpHandler) Connect(w http.ResponseWriter, r *http.Request) Response {

	gatewayConfig, err := decode[mcp.ServerGatewayConfig](r)
	if err != nil {
		return NewInvalidRequest(err)
	}

	id, err := h.mcpSvc.Connect(r.Context(), gatewayConfig)
	if err != nil {
		var validationErr *types.ValidationError
		if errors.As(err, &validationErr) {
			return NewInvalidRequest(err)
		}
		if errors.Is(err, types.ErrAlreadyExist) {
			return NewBadRequest("already exist")
		}

		return NewInternalError(err)
	}

	return NewJSONResponse(http.StatusOK, map[string]string{
		"created_id": string(id),
	})
}

// DELETE /mcp/{id}
func (h *mcpHandler) Disconnect(w http.ResponseWriter, r *http.Request) Response {
	mcpID := r.PathValue("id")

	if err := h.mcpSvc.Disconnect(mcp.MCPServerID(mcpID)); err != nil {
		if errors.Is(err, types.ErrIsNotExist) {
			return NewBadRequest(err.Error())
		}
		return NewInternalError(err)
	}
	return NewResponse(http.StatusOK)
}

// POST /mcp/reload
func (h *mcpHandler) Reload(w http.ResponseWriter, r *http.Request) Response {
	if err := h.mcpSvc.Reload(r.Context()); err != nil {
		return NewInternalError(err)
	}
	return NewResponse(http.StatusOK)
}

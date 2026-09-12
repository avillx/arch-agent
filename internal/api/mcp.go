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

	type MCPServerDTO struct {
		Config mcp.ServerGatewayConfig `json:"config"`
		Tools  ToolReprDTO             `json:"tools"`
	}

	type MCPServersDTO map[mcp.MCPServerID]MCPServerDTO

	mcpServers := MCPServersDTO{}
	for _, srv := range h.mcpSvc.List() {

		tools := ToolReprDTO{}
		for _, t := range srv.Tools() {
			tools[t.Name()] = t.Description()
		}

		mcpServers[srv.ID()] = MCPServerDTO{
			Config: srv.Config(),
			Tools:  tools,
		}
	}

	return NewJSONResponse(http.StatusOK, mcpServers)
}

// POST /mcp/{mcp}
func (h *mcpHandler) Connect(w http.ResponseWriter, r *http.Request) Response {

	mcpID := mcp.MCPServerID(r.PathValue("mcp"))

	gatewayConfig, err := decode[mcp.ServerGatewayConfig](r)
	if err != nil {
		return NewInvalidRequest(err)
	}

	if err := h.mcpSvc.SetServer(r.Context(), mcpID, gatewayConfig); err != nil {
		// if mcp server is not starting well so problem in config
		// that's the reason of invalid request
		return NewInvalidRequest(err)
	}

	return NewResponse(http.StatusOK)
}

// DELETE /mcp/{mcp}
func (h *mcpHandler) Disconnect(w http.ResponseWriter, r *http.Request) Response {
	mcpID := mcp.MCPServerID(r.PathValue("mcp"))
	if err := h.mcpSvc.DeleteServer(mcpID); err != nil {
		if errors.Is(err, types.ErrIsNotExist) {
			return NewBadRequest(err.Error())
		}
		return NewInternalError(err)
	}
	return NewResponse(http.StatusOK)
}

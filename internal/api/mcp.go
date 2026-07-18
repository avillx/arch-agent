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

	dto := map[string]any{}

	for _, mcpServers := range h.mcpSvc.List() {

		toolList := []string{}
		for _, t := range mcpServers.Tools() {
			toolList = append(toolList, string(t.Name()))
		}

		dto[string(mcpServers.ID())] = map[string]any{
			"transport": mcpServers.Gateway().Type(),
			"tools":     toolList,
		}
	}

	return respond(w, http.StatusOK, map[string]any{"mcp_servers": dto})
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
	return respond(w, http.StatusOK, map[string]string{"msg": "success"})
}

// POST /api/v1/reload
func (h *mcpHandler) Reload(w http.ResponseWriter, r *http.Request) error {
	if err := h.mcpSvc.Reload(r.Context()); err != nil {
		return internal(err)
	}
	return respond(w, http.StatusOK, map[string]string{"msg": "success"})
}

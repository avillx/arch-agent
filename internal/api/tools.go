package api

import (
	"arch-agent/internal/agent"
	"arch-agent/internal/tools"
	"net/http"
)

// handler
type toolsHandler struct {
	toolSvc *tools.Service
}

type ToolReprDTO map[agent.ToolName]string
type ToolServerReprDTO map[string]ToolReprDTO

// GET /tools
func (h *toolsHandler) List(w http.ResponseWriter, _ *http.Request) Response {

	toolServers := h.toolSvc.AllToolServers()

	toolServersDTO := ToolServerReprDTO{}
	for name, s := range toolServers {
		tools := ToolReprDTO{}
		for _, t := range s.Tools() {
			tools[t.Name()] = t.Description()
		}
		toolServersDTO[name] = tools
	}

	return NewJSONResponse(http.StatusOK, toolServersDTO)
}

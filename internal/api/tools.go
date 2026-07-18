package api

import (
	"arch-agent/internal/tools"
	"net/http"
)

// handler
type toolsHandler struct {
	toolSvc *tools.Service
}

// GET /tools
func (h *toolsHandler) List(w http.ResponseWriter, _ *http.Request) error {

	toolServers := h.toolSvc.AllToolServers()

	list := map[string][]string{}
	for name, s := range toolServers {
		tools := []string{}
		for _, t := range s.Tools() {
			tools = append(tools, string(t.Name()))
		}

		list[name] = tools
	}

	return respond(w, http.StatusOK, map[string]any{"tool_servers": list})
}

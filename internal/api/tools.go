package api

import (
	"arch-agent/internal/tools"
	"net/http"
)

// handler
type toolsHandler struct {
	toolSvc *tools.Service
}

type ToolReprDTO struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type ToolServerReprDTO struct {
	Name  string        `json:"name"`
	Tools []ToolReprDTO `json:"tools"`
}

// GET /tools
func (h *toolsHandler) List(w http.ResponseWriter, _ *http.Request) Response {

	toolServers := h.toolSvc.AllToolServers()

	toolServersDTO := []ToolServerReprDTO{}
	for name, s := range toolServers {
		tools := []ToolReprDTO{}
		for _, t := range s.Tools() {
			tools = append(tools, ToolReprDTO{
				Name:        string(t.Name()),
				Description: t.Description(),
			})
		}
		toolServersDTO = append(toolServersDTO, ToolServerReprDTO{
			Name:  name,
			Tools: tools,
		})
	}

	return NewJSONResponse(http.StatusOK, toolServersDTO)
}

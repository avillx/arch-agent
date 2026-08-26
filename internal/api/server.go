package api

import (
	"arch-agent/internal/agent"
	"arch-agent/internal/chat"
	"arch-agent/internal/mcp"
	"arch-agent/internal/memory"
	"arch-agent/internal/model"
	"arch-agent/internal/session"
	"arch-agent/internal/task"
	"arch-agent/internal/tools"
	"encoding/json"
	"log/slog"
	"net/http"
)

// Responder
type HTTPServer struct {
	*http.ServeMux
	logger           *slog.Logger
	taskSvc          *task.Service
	chatSvc          *chat.Dispatcher
	sessSvc          *session.Service
	toolsSvc         *tools.Service
	mcpSvc           *mcp.Service
	memoryRepo       agent.MemoryRepo
	memoryIndexer    agent.MemoryIndexer
	consolidationSvc *memory.ConsolidationService
	activityStore    activityStore
	agentRepo        agent.Repo
	providerSvc      *model.ProviderService
	idGen            IDGenerator
}

func NewHTTPServer(
	logger *slog.Logger,
	taskSvc *task.Service,
	chatSvc *chat.Dispatcher,
	sessSvc *session.Service,
	toolsSvc *tools.Service,
	mcpSvc *mcp.Service,
	memoryRepo agent.MemoryRepo,
	memoryIndexer agent.MemoryIndexer,
	consolidationSvc *memory.ConsolidationService,
	activityStore activityStore,
	agentRepo agent.Repo,
	providerSvc *model.ProviderService,
	idGen IDGenerator,
) *HTTPServer {

	srv := &HTTPServer{
		logger:           logger.WithGroup("api"),
		ServeMux:         http.NewServeMux(),
		taskSvc:          taskSvc,
		chatSvc:          chatSvc,
		sessSvc:          sessSvc,
		toolsSvc:         toolsSvc,
		mcpSvc:           mcpSvc,
		memoryRepo:       memoryRepo,
		memoryIndexer:    memoryIndexer,
		consolidationSvc: consolidationSvc,
		activityStore:    activityStore,
		agentRepo:        agentRepo,
		providerSvc:      providerSvc,
		idGen:            idGen,
	}

	srv.registerRoutes()

	return srv
}

func (s *HTTPServer) registerRoutes() {
	taskHandler := &taskHandler{taskSvc: s.taskSvc}
	s.HandleFunc("GET /task", taskHandler.List)
	s.HandleFunc("POST /task/{name}", taskHandler.Create)
	s.HandleFunc("DELETE /task/{name}", taskHandler.Delete)
	s.HandleFunc("PATCH /task/{name}", taskHandler.Patch)

	sessHandler := &sessionHandler{sessSvc: s.sessSvc}
	s.HandleFunc("GET /session/{agent}", sessHandler.List)
	s.HandleFunc("GET /session/{agent}/{session_id}", sessHandler.Get)
	s.HandleFunc("POST /session/{agent}", sessHandler.Create)
	s.HandleFunc("DELETE /session/{agent}/{session_id}", sessHandler.Delete)

	toolsHandler := &toolsHandler{toolSvc: s.toolsSvc}
	s.HandleFunc("GET /tools", toolsHandler.List)

	mcpHandler := &mcpHandler{mcpSvc: s.mcpSvc}
	s.HandleFunc("GET /mcp", mcpHandler.List)
	s.HandleFunc("POST /mcp", mcpHandler.Connect)
	s.HandleFunc("POST /mcp/reload", mcpHandler.Reload)
	s.HandleFunc("DELETE /mcp/{id}", mcpHandler.Disconnect)

	memoryHandler := NewMemoryHandler(s.consolidationSvc, s.memoryIndexer, s.memoryRepo)
	s.HandleFunc("POST /memory/{agent}/consolidate", memoryHandler.Consolidate)
	s.HandleFunc("GET /memory/{agent}/{memory_name}", memoryHandler.Get)
	s.HandleFunc("GET /memory/{agent}", memoryHandler.List)

	provToolHandler := NewProvidedToolsRouter(s.idGen)
	s.HandleFunc("POST /toolresult/{id}", provToolHandler.ResolveCall)

	chatHandler := &chatHandler{provToolRegister: provToolHandler, chatDispatcher: s.chatSvc}
	s.HandleFunc("POST /chat/{agent}/{session}", chatHandler.Chat)
	s.HandleFunc("POST /chat/{agent}/{session}/interrupt", chatHandler.Interrupt)

	activityHandler := &activityHandler{store: s.activityStore}
	s.HandleFunc("GET /activity", activityHandler.Activity)

	agentHandler := &agentHandler{repo: s.agentRepo}
	s.HandleFunc("GET /agent", agentHandler.List)
	s.HandleFunc("GET /agent/{id}", agentHandler.Read)
	s.HandleFunc("POST /agent/{id}", agentHandler.Create)
	s.HandleFunc("PUT /agent/{id}", agentHandler.Update)
	s.HandleFunc("DELETE /agent/{id}", agentHandler.Delete)

	providerHandler := &providerHandler{providerSvc: s.providerSvc}
	s.HandleFunc("GET /providers", providerHandler.GetAll)
	s.HandleFunc("GET /providers/{name}", providerHandler.GetProvider)
	s.HandleFunc("POST /providers", providerHandler.AddProvider)
	s.HandleFunc("PATCH /providers/{name}", providerHandler.UpdateProvider)
	s.HandleFunc("DELETE /providers/{name}", providerHandler.DeleteProvider)

	// models
	s.HandleFunc("POST /providers/{name}/models/{model}", providerHandler.SetModel)
	s.HandleFunc("DELETE /providers/{name}/models/{model}", providerHandler.DeleteModel)

}

func (s *HTTPServer) HandleFunc(pattern string, handler func(http.ResponseWriter, *http.Request) Response) {
	s.ServeMux.HandleFunc(pattern, s.wrapResponse(handler))
}

func (s *HTTPServer) wrapResponse(h func(w http.ResponseWriter, r *http.Request) Response) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger := s.logger.With("method", r.Method, "path", r.URL.Path)
		logger.Info("incoming request")

		response := h(w, r)

		// log error
		if err, ok := response.(error); ok {
			logger.Error("internal", "error", err)
		}

		// silent case. no response provided
		if response == nil {
			return
		}

		logger.Info("respond", "status", response.StatusCode())

		// respond json
		if jr, ok := response.(JSONResponse); ok {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(jr.StatusCode())
			enc := json.NewEncoder(w)
			if err := enc.Encode(jr.Content()); err != nil {
				logger.Error("response encoding", "error", err)
			}
			return
		}

		// respond status code
		w.WriteHeader(response.StatusCode())
	}
}

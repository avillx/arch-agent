package service

type App struct {
	A2A *A2AService
	// LiveChatSvc    *LiveChatService
	SessionChatSvc *SessionChatService
	AgentSvc       *AgentService
}

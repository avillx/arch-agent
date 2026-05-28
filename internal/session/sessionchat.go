package session

// TODO: more flexable token limit check
// a no one agent can't really efficent work with 100k+ context at this moment
// is a temporary solution
const DefaultTokenLimit = 100_000

// type SessionChatService struct {
// 	chatSvc        agent.ChatSvc
// 	sessionService *SessionService
// }

// func NewSessionChatService(
// 	chatSvc agent.ChatSvc,
// 	sessionService *SessionService,
// ) *SessionChatService {
// 	return &SessionChatService{
// 		chatSvc:        chatSvc,
// 		sessionService: sessionService,
// 	}
// }

// func (s *SessionChatService) SessionChat(
// 	ctx context.Context,
// 	agentID agent.ID,
// 	sessionID agent.SessionID,
// 	presummaryAdditioanlSystemPrompt string,
// 	postsummaryAdditioanlSystemPrompt string,
// 	request string,
// 	onResult func(result *agent.ReasonResult),
// ) error {
// 	slog.Debug("Session chat", "agent", agentID, "request", request)

// 	sess, err := s.sessionService.Get(agentID, sessionID)
// 	if err != nil {
// 		return err
// 	}

// 	summaries := sess.Summary()
// 	if summaries != "" {
// 		summaries = strings.Join([]string{
// 			"<summary>",
// 			"# Here's previus conversation summary:",
// 			summaries,
// 			"</summary>",
// 		}, "\n")
// 	}

// 	additionalSystemPrompt := strings.Join([]string{
// 		presummaryAdditioanlSystemPrompt,
// 		summaries,
// 		postsummaryAdditioanlSystemPrompt}, "\n")

// 	userMessages := []agent.Message{agent.NewUserMessage(request)}
// 	newMessages, err := s.chatSvc.Chat(
// 		ctx,
// 		agentID,
// 		additionalSystemPrompt,
// 		append(sess.Messages(), userMessages...),
// 		onResult,
// 		nil,
// 	)
// 	if err != nil {
// 		return err
// 	}

// 	sess.AddMessages(userMessages)
// 	sess.AddMessages(newMessages)

// 	if IsSessionOverflow(sess) {
// 		if err := s.truncateSession(ctx, sess); err != nil {
// 			slog.Error("truncate session", "agent", agentID, "session", sess.ID(), "error", err)
// 		}
// 	}

// 	return s.sessionService.Save(agentID, sess)
// }

// const thereshold float32 = 0.9

func IsShouldCompact(sess Session, contextLimit int) bool {

	if sess.Tokens() >= (contextLimit / 100 * 90) {
		return true
	}

	return false
}

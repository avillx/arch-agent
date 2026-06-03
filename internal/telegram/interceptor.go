package telegram

// type A2AInterceptor struct {
// 	groupID int64
// 	orch    *BotOrchestrator
// 	callCh  <-chan a2a.Call
// 	respCh  <-chan a2a.Response

// 	requestMessageQueue map[a2a.Call]tgbotapi.Message
// 	mu                  sync.RWMutex
// }

// func NewA2AInterceptor(groupID int64, orch *BotOrchestrator, a2aService *a2a.Service) *A2AInterceptor {
// 	return &A2AInterceptor{
// 		groupID: groupID,
// 		orch:    orch,
// 		callCh:  a2aService.CallChannel(),
// 		respCh:  a2aService.ResponseChannel(),

// 		requestMessageQueue: make(map[a2a.Call]tgbotapi.Message),
// 	}
// }

// func (i *A2AInterceptor) CollectGarbage() {
// 	i.mu.Lock()
// 	defer i.mu.Unlock()

// 	for k, v := range i.requestMessageQueue {
// 		if isGarbage(v) {
// 			delete(i.requestMessageQueue, k)
// 		}
// 	}
// }

// func (i *A2AInterceptor) Run(ctx context.Context) {

// 	ticker := time.NewTicker(1 * time.Minute)
// 	defer ticker.Stop()

// 	for {
// 		select {
// 		case <-ticker.C:
// 			i.CollectGarbage()
// 		case <-ctx.Done():
// 			return
// 		case call := <-i.callCh:
// 			if err := i.onA2ACall(ctx, call); err != nil {
// 				slog.Error("call resolve", "error", err)
// 			}
// 		case resp := <-i.respCh:
// 			if err := i.onA2AResponse(ctx, resp); err != nil {
// 				slog.Error("response resolve", "error", err)
// 			}
// 		}
// 	}
// }

// func (i *A2AInterceptor) onA2ACall(_ context.Context, c a2a.Call) error {
// 	bot, err := i.orch.Get(c.Caller)
// 	if err != nil {
// 		// only not exist errors.
// 		// intercept only bot provided agent. if agent hasn't its not error
// 		return nil
// 	}

// 	msgs, err := bot.SendMessage(i.groupID, fmt.Sprintf("@%s\n%s", c.Recivier, c.Request), 0)

// 	if len(msgs) > 0 {
// 		i.mu.Lock()
// 		i.requestMessageQueue[c] = msgs[0]
// 		i.mu.Unlock()
// 	}

// 	return err
// }

// func (i *A2AInterceptor) onA2AResponse(_ context.Context, r a2a.Response) error {

// 	bot, err := i.orch.Get(r.Recivier)
// 	if err != nil {
// 		// only not exist errors.
// 		// intercept only bot provided agent. if agent hasn't its not error
// 		return nil
// 	}

// 	text := r.Response
// 	messageID := 0

// 	i.mu.RLock()
// 	msg, ok := i.requestMessageQueue[r.Call]
// 	i.mu.RUnlock()

// 	if ok {
// 		messageID = msg.MessageID
// 	} else {
// 		text = fmt.Sprintf("@%s %s", r.Caller, text)
// 	}

// 	_, err = bot.SendMessage(i.groupID, text, messageID)

// 	i.mu.Lock()
// 	defer i.mu.Unlock()
// 	delete(i.requestMessageQueue, r.Call)

// 	return err
// }

// func isGarbage(msg tgbotapi.Message) bool {
// 	return time.Now().After(msg.Time().Add(1 * time.Minute))
// }

package service

import (
	"arch-agent/internal/domain/activity"
	"arch-agent/internal/domain/agent"
	"arch-agent/internal/domain/session"
	"arch-agent/internal/domain/task"
	"arch-agent/internal/domain/types"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"
)

const LiveChatSessionID = "live_chat"

type ActivityRepo interface {
	Log(agent.ID, activity.Record) error
	GetActivity(agent.ID, time.Time) (string, error)
}

type ChatService struct {
	agentService   *AgentService
	sessionService *SessionService
	taskService    *TaskService
	activityRepo   ActivityRepo
}

func NewChatService(
	agentService *AgentService,
	sessionService *SessionService,
	taskService *TaskService,
	activityRepo ActivityRepo,
) *ChatService {
	return &ChatService{
		agentService:   agentService,
		sessionService: sessionService,
		taskService:    taskService,
		activityRepo:   activityRepo,
	}
}

func (s *ChatService) SessionChat(
	ctx context.Context,
	agentID agent.ID,
	sessionID session.ID,
	request string,
	onResult func(result *agent.ReasonResult),
) error {
	slog.Debug("Session chat", "agent", agentID, "request", request)

	sess, err := s.sessionService.Get(agentID, sessionID)
	if err != nil {
		return err
	}

	userMessages := []types.Message{types.NewUserMessage(request)}
	newMessages, err := s.chat(
		ctx,
		agentID,
		nil,
		sess.Messages(),
		userMessages,
		onResult,
	)
	if err != nil {
		return err
	}

	sess.AddMessages(userMessages)
	sess.AddMessages(newMessages)

	return s.sessionService.Save(agentID, sess)
}

func (s *ChatService) LiveSessionChat(
	ctx context.Context,
	agentID agent.ID,
	request string,
	onResult func(result *agent.ReasonResult),
) error {
	slog.Debug("live session chat", "agent", agentID, "request", request)
	s.updateDropLiveSessionTask(ctx, agentID)

	sess, err := s.sessionService.Get(agentID, LiveChatSessionID)
	if err != nil {
		return err
	}

	memoryMessage, err := s.memoryMessage(agentID)
	if err != nil {
		return err
	}

	userMessages := []types.Message{types.NewUserMessage(request)}
	newMessages, err := s.chat(
		ctx,
		agentID,
		[]types.Message{memoryMessage},
		sess.Messages(),
		userMessages,
		onResult,
	)
	if err != nil {
		return err
	}

	sess.AddMessages(userMessages)
	sess.AddMessages(newMessages)

	return s.sessionService.Save(agentID, sess)
}

// drop chat and log it to activity
func (s *ChatService) dropLiveSession(
	ctx context.Context,
	agentID agent.ID,
) error {
	slog.Debug("live session drop initiated", "agent", agentID)
	sess, err := s.sessionService.Get(agentID, LiveChatSessionID)
	if err != nil {
		return err
	}

	newMessages, err := s.chat(
		ctx,
		agentID,
		nil,
		sess.Messages(),
		[]types.Message{types.NewUserMessage(Memorize())},
		nil,
	)
	if err != nil {
		return err
	}

	logContent := newMessages[len(newMessages)-1].Content()

	if err := s.activityRepo.Log(agentID, activity.NewRecord(logContent)); err != nil {
		return err
	}

	s.sessionService.Delete(agentID, LiveChatSessionID)

	return nil
}

func (s *ChatService) chat(
	ctx context.Context,
	agentID agent.ID,
	preContextMessages []types.Message,
	contextMessages []types.Message,
	postContextMessages []types.Message,
	onResult func(result *agent.ReasonResult),
) (newMsgs []types.Message, err error) {

	a, err := s.agentService.GetAgent(agentID)
	if err != nil {
		return nil, err
	}

	a.OnResult(onResult)

	conversation := []types.Message{}
	if preContextMessages != nil {
		conversation = append(conversation, preContextMessages...)
	}

	if contextMessages != nil {
		conversation = append(conversation, contextMessages...)
	}

	if postContextMessages != nil {
		conversation = append(conversation, postContextMessages...)
	}

	return a.Chat(ctx, conversation)
}

// update task for drop live chat
// creates a task if it is not exist
func (s *ChatService) updateDropLiveSessionTask(ctx context.Context, agentID agent.ID) {
	// override old task if its exits
	s.taskService.AddTask(
		ctx,
		dropChatTaskID(agentID),
		s.createDropChatTask(agentID),
	)
}

// create a new task for drop live chat
func (s *ChatService) createDropChatTask(agentID agent.ID) *task.Task {
	return task.NewTask(
		// reglament
		task.Every{D: 3 * time.Minute},

		// description
		fmt.Sprintf("drop live chat %s", agentID),

		// execution
		func(ctx context.Context, t *task.Task) {
			if err := s.dropLiveSession(ctx, agentID); err != nil {
				slog.Error("drop chat", "error", err)
			}
			t.Stop()
		},
	)
}

// Creates a message with activity for agent
func (s *ChatService) memoryMessage(agentID agent.ID) (*types.UserMessage, error) {
	today, err := s.activityRepo.GetActivity(agentID, time.Now())
	if err != nil {
		if !errors.Is(err, activity.ErrNoActivity) {
			return nil, err
		}
	}
	yesterday, err := s.activityRepo.GetActivity(agentID, time.Now().Local().AddDate(0, 0, -1))
	if err != nil {
		if !errors.Is(err, activity.ErrNoActivity) {
			return nil, err
		}
	}

	return types.NewUserMessage(Memory(today, yesterday)), nil
}

// Creates a name (ID) for task to drop agent live session
func dropChatTaskID(agentID agent.ID) string {
	return fmt.Sprintf("drop_%s_live_chat", agentID)
}

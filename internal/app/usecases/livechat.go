package usecases

import (
	service "arch-agent/internal/app"
	"fmt"
	"log/slog"

	"arch-agent/internal/domain/activity"
	"arch-agent/internal/domain/agent"
	"arch-agent/internal/domain/session"
	"arch-agent/internal/domain/task"
	"arch-agent/internal/domain/types"
	"context"
	"errors"
	"time"
)

const LiveChatSessionID = "live_chat"

type ActivityRepo interface {
	Log(agent.ID, activity.Record) error
	GetActivity(agent.ID, time.Time) (string, error)
}

type LiveChat struct {
	agentService   *service.AgentService
	sessionService *service.SessionService
	taskService    *service.TaskService
	activityRepo   ActivityRepo
}

func (uc *LiveChat) Chat(
	ctx context.Context,
	sessionID session.ID,
	agentID agent.ID,
	request string,
	onContent func(string),
) error {

	a, err := uc.agentService.GetAgent(agentID)
	if err != nil {
		return err
	}

	sess, err := uc.sessionService.Get(agentID, LiveChatSessionID)
	if err != nil {
		return err
	}

	userMsg := types.NewUserMessage(request)

	memoryMessage, err := uc.memoryMessage(agentID)
	if err != nil {
		return err
	}

	conversation := []types.Message{}
	conversation = append(conversation, memoryMessage)
	conversation = append(conversation, sess.Messages()...)
	conversation = append(conversation, userMsg)

	a.OnContent(onContent)

	var errc error
	newMsgs, err := a.Chat(ctx, conversation)
	if err != nil {
		errc = errors.Join(errc, err)
	}

	sess.AddMessages(append([]types.Message{userMsg}, newMsgs...))

	if err := uc.sessionService.Save(agentID, sess); err != nil {
		errc = errors.Join(errc, err)
	}

	return errc
}

// drop chat and log it to activity
func (uc *LiveChat) dropChat(ctx context.Context, agentID agent.ID) error {

	a, err := uc.agentService.GetAgent(agentID)
	if err != nil {
		return err
	}

	sess, err := uc.sessionService.Get(agentID, LiveChatSessionID)
	if err != nil {
		return err
	}

	conversation := []types.Message{}
	conversation = append(conversation, sess.Messages()...)
	conversation = append(conversation, types.NewUserMessage(service.Memorize()))

	newMsgs, err := a.Chat(ctx, conversation)
	if err != nil {
		return err
	}

	logContent := newMsgs[len(newMsgs)-1].Content()

	if err := uc.activityRepo.Log(agentID, activity.NewRecord(logContent)); err != nil {
		return err
	}

	// uc.sessionService.Delete(agentID, "live_chat")
	return nil
}

// update task for drop live chat
// creates a task if it is not exist
func (uc *LiveChat) updateDropChatTask(ctx context.Context, agentID agent.ID) error {
	taskID := dropChatTaskID(agentID)

	dropTask, err := uc.taskService.Task(taskID)
	if errors.Is(types.ErrIsNotExist, err) {
		dropTask = uc.createDropChatTask(agentID, taskID)
		uc.taskService.AddTask(ctx,
			taskID,
			dropTask,
		)
		return nil
	}
	if err != nil {
		return err
	}

	dropTask.SetReglament(
		task.ExecuteAt(time.Now().Add(10 * time.Minute)),
	)

	return nil
}

// create a new task for drop live chat
func (uc *LiveChat) createDropChatTask(agentID agent.ID, taskID string) *task.Task {
	return task.NewTask(
		// reglament
		task.ExecuteAt(time.Now().Add(10*time.Minute)),

		// description
		fmt.Sprintf("drop live chat %s", taskID),

		// execution
		func(ctx context.Context) {
			if err := uc.dropChat(ctx, agentID); err != nil {
				slog.Error("drop chat", "error", err)
			}
			if err := uc.taskService.RemoveTask(taskID); err != nil {
				slog.Error("drop chat remove task", "error", err)
			}
		},
	)
}

// Creates a message with activity for agent
func (uc *LiveChat) memoryMessage(agentID agent.ID) (*types.UserMessage, error) {
	yesterday, err := uc.activityRepo.GetActivity(agentID, time.Now())
	if err != nil {
		return nil, err
	}
	today, err := uc.activityRepo.GetActivity(agentID, time.Now().Local().AddDate(0, 0, -1))
	if err != nil {
		return nil, err
	}

	return types.NewUserMessage(service.Memory(yesterday, today)), nil
}

// Creates a name (ID) for task to drop agent live session
func dropChatTaskID(agentID agent.ID) string {
	return fmt.Sprintf("drop_live_chat_%s", agentID)
}

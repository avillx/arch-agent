package task

import (
	"arch-agent/internal/agent"
	"arch-agent/internal/prompt"
	"arch-agent/internal/types"
	"context"
	"fmt"
	"log/slog"
	"sync"
)

type UUIDGenerator interface {
	New() string
}

// runtime
type TaskRuntime struct {
	runningTasks map[string]*RunningTask
	mu           sync.RWMutex
	done         chan string
}

func NewTaskRuntime() *TaskRuntime {
	return &TaskRuntime{
		runningTasks: map[string]*RunningTask{},
		done:         make(chan string),
	}
}

func (r *TaskRuntime) Spawn(id string, t Task, exec func(Task)) {
	r.mu.Lock()

	if runningTask, ok := r.runningTasks[id]; ok {
		runningTask.Stop()
	}

	newTask := NewRunningTask(t, exec)
	r.runningTasks[id] = newTask
	r.mu.Unlock()

	go func() {

		newTask.Start()

		r.mu.Lock()
		if runningTask, ok := r.runningTasks[id]; ok {
			runningTask.Stop()
			delete(r.runningTasks, id)
		}
		r.mu.Unlock()

		r.done <- id
	}()
}

func (r *TaskRuntime) Kill(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if runningTask, ok := r.runningTasks[id]; ok {
		runningTask.Stop()
		return nil
	}

	return types.ErrIsNotExist
}

func (r *TaskRuntime) Done() chan string {
	return r.done
}

type taskExecutor struct {
	agentSvc     agent.AgentService
	chatSvc      agent.ChatSvc
	activityRepo agent.ActivityRepo
}

// executor
func NewTaskExecutor(
	agentSvc agent.AgentService,
	chatSvc agent.ChatSvc,
	activityRepo agent.ActivityRepo,
) *taskExecutor {
	return &taskExecutor{
		agentSvc:     agentSvc,
		chatSvc:      chatSvc,
		activityRepo: activityRepo,
	}
}

func (s *taskExecutor) Validate(t Task) error {
	// get existed agents
	agentsCfgs, err := s.agentSvc.List()
	if err != nil {
		return err
	}

	// validate agents
	agentMap := map[agent.ID]struct{}{}
	for _, cfg := range agentsCfgs {
		agentMap[cfg.ID()] = struct{}{}
	}
	for _, agent := range t.Recipients {
		if _, ok := agentMap[agent]; !ok {
			return fmt.Errorf("agent %s is not exist", agent)
		}
	}

	return nil
}

func (s *taskExecutor) Executor(t Task) {
	for _, r := range t.Recipients {
		slog.Info("processing task", "agent", r, "task", t.Name)
		if err := s.processRecipientTask(agent.ID(r), t.Name, t.Request); err != nil {
			slog.Error("task processing", "agent", r, "task", t.Name, "error", err)
		}
	}
}

func (s *taskExecutor) processRecipientTask(agentID agent.ID, taskName, request string) error {
	const autonomusWorking = "Now You working autonomusly if somthing wrong try to contact with someone"

	ctx := context.Background()

	history, err := s.chatSvc.Chat(
		ctx,
		agentID,
		autonomusWorking,
		[]agent.Message{
			agent.NewUserMessage(request),
		},
		nil,
		nil,
	)
	if err != nil {
		return err
	}

	report := ""

	_, err = s.chatSvc.Chat(
		ctx,
		agentID,
		autonomusWorking,
		append(history, agent.NewUserMessage(prompt.Report())),
		func(result *agent.ReasonResult) {
			report += result.Content
		},
		nil,
	)
	if err != nil {
		return err
	}

	if err := s.activityRepo.Log(
		agentID,
		agent.NewRecord(
			fmt.Sprintf("Task: %s\nReport:\n%s", taskName, report),
		),
	); err != nil {

		return err
	}

	return nil
}

type TaskRecord struct {
	Active bool
	Task
}

type TaskRepo interface {
	All() (map[string]*TaskRecord, error)
	Get(id string) (*TaskRecord, error)
	Save(id string, t *TaskRecord) error
	Delete(id string) error
}

// service
type TaskService struct {
	runtime  *TaskRuntime
	executor *taskExecutor
	repo     TaskRepo
}

func NewTaskService(
	ctx context.Context,
	repo TaskRepo,
	executor *taskExecutor,
) (*TaskService, error) {

	s := &TaskService{
		repo:     repo,
		runtime:  NewTaskRuntime(),
		executor: executor,
	}

	s.processDoneTasks(ctx)

	recs, err := repo.All()
	if err != nil {
		return nil, err
	}

	for id, rec := range recs {
		if !rec.Active {
			continue
		}
		// TODO validate task on service creation
		// !Caution it may have edge case issues if agent is not exist
		// validation return error and program not started
		if err := s.Start(id); err != nil {
			return nil, err
		}
	}

	return s, nil
}

func (s *TaskService) All() (map[string]*TaskRecord, error) {
	return s.repo.All()
}

func (s *TaskService) New(t Task) (string, error) {

	// validate
	if err := s.executor.Validate(t); err != nil {
		return "", err
	}

	// save to repo
	if err := s.repo.Save(t.Name, &TaskRecord{Active: false, Task: t}); err != nil {
		return "", err
	}

	return t.Name, nil
}

func (s *TaskService) Start(id string) error {

	rec, err := s.repo.Get(id)
	if err != nil {
		return err
	}

	s.runtime.Spawn(id, rec.Task, s.executor.Executor)
	rec.Active = true

	return s.repo.Save(id, rec)
}

func (s *TaskService) Stop(id string) error {
	return s.runtime.Kill(id)
}

func (s *TaskService) Delete(id string) error {
	s.runtime.Kill(id)

	return s.repo.Delete(id)
}

// blocking
func (s *TaskService) processDoneTasks(ctx context.Context) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case doneTaskID := <-s.runtime.Done():
				rec, err := s.repo.Get(doneTaskID)
				if err != nil {
					slog.Error("process done tasks", "task", doneTaskID, "error", err)
				}

				rec.Active = false

				if err := s.repo.Save(doneTaskID, rec); err != nil {
					slog.Error("process done tasks", "task", doneTaskID, "error", err)
				}
			}
		}
	}()
}

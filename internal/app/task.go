package service

import (
	"arch-agent/internal/domain/activity"
	"arch-agent/internal/domain/agent"
	"arch-agent/internal/domain/task"
	"arch-agent/internal/domain/types"
	"context"
	"fmt"
	"log/slog"
	"sync"
)

// runtime
type TaskRuntime struct {
	runningTasks map[string]*task.RunningTask
	mu           sync.RWMutex
	done         chan string
}

func NewTaskRuntime() *TaskRuntime {
	return &TaskRuntime{
		runningTasks: map[string]*task.RunningTask{},
		done:         make(chan string),
	}
}

func (r *TaskRuntime) Spawn(id string, t task.Task, exec func(task.Task)) {
	r.mu.Lock()

	if runningTask, ok := r.runningTasks[id]; ok {
		runningTask.Stop()
	}

	newTask := task.NewRunningTask(t, exec)
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
	agentService *AgentService
	activityRepo ActivityRepo
}

// executor
func NewTaskExecutor(
	agentService *AgentService,
	activityRepo ActivityRepo,
) *taskExecutor {
	return &taskExecutor{
		agentService: agentService,
		activityRepo: activityRepo,
	}
}

func (s *taskExecutor) Executor(t task.Task) {
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

	history, err := s.agentService.Chat(
		ctx,
		agentID,
		autonomusWorking,
		[]types.Message{
			types.NewUserMessage(request),
		},
		nil,
		nil,
		nil,
	)
	if err != nil {
		return err
	}

	report := ""

	_, err = s.agentService.Chat(
		ctx,
		agentID,
		autonomusWorking,
		history,
		[]types.Message{
			types.NewUserMessage(Report()),
		},
		nil,
		func(result *agent.ReasonResult) {
			report += result.Content
		},
	)
	if err != nil {
		return err
	}

	if err := s.activityRepo.Log(
		agentID,
		activity.NewRecord(
			fmt.Sprintf("Task: %s\nReport:\n%s", taskName, report),
		),
	); err != nil {

		return err
	}

	return nil
}

type TaskRecord struct {
	Active bool
	task.Task
}

type TaskRepo interface {
	All() (map[string]*TaskRecord, error)
	Get(id string) (*TaskRecord, error)
	Save(id string, t *TaskRecord) error
	Delete(id string) error
}

// service
type TaskService struct {
	uuidGen  UUIDGenerator
	runtime  *TaskRuntime
	executor *taskExecutor
	repo     TaskRepo
}

func NewTaskService(
	ctx context.Context,
	uuidGen UUIDGenerator,
	repo TaskRepo,
	executor *taskExecutor,
) (*TaskService, error) {

	s := &TaskService{
		uuidGen:  uuidGen,
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

		if err := s.Start(id); err != nil {
			return nil, err
		}
	}

	return s, nil
}

func (s *TaskService) All() (map[string]*TaskRecord, error) {
	return s.repo.All()
}

func (s *TaskService) New(t task.Task) (string, error) {
	uuid := s.uuidGen.New()
	if err := s.repo.Save(uuid, &TaskRecord{Active: false, Task: t}); err != nil {
		return "", err
	}

	return uuid, nil
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

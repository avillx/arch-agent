package files

import (
	"arch-agent/internal/agent"
	"arch-agent/internal/sentinel"
	"arch-agent/internal/task"
	"arch-agent/internal/types"
	"bytes"
	"context"
	"fmt"
	"os"
	"sync"

	"github.com/fsnotify/fsnotify"
	toml "github.com/pelletier/go-toml/v2"
)

const TaskConfigFile = "tasks.toml"
const taskConfigDoc = `# Cron tasks config

# Tasks are planned requests sent to agent recipients on a cron schedule.
# When a recipient receives a request, it works autonomously.

# Task example:

# unique task name
# [task_name]

# Short, one line description of task
# description="short task description"

# Agent recipients
# recipients=["agent_id1","agent_id2"]

# Cron schedule
# schedule="* * * * *"

# Active field defines the state
# * true - task enabled
# * false - task disabled
# active=true

# When once is true, the task will disappear after its first execution
# once=true

# Exhaustive request that the recipient will receive on schedule
# Prefer use multiline """...""" format over one line '...' 
# request="""
# Use your skill /some/path
# do ...
# Confirm result
# """

# Do not touch this comment!
# After edit, ensure file consistency and comment integrity`

type TaskFiles struct {
	mu sync.RWMutex
	fs *FileSystem
}

func NewTaskFiles(fs *FileSystem) (*TaskFiles, error) {

	if err := ensureFilePlaceholder(fs, TaskConfigFile, []byte(taskConfigDoc)); err != nil {
		return nil, err
	}

	return &TaskFiles{
		fs: fs,
	}, nil
}

func (tf *TaskFiles) All() (map[string]task.TaskConfig, error) {
	tf.mu.RLock()
	defer tf.mu.RUnlock()

	return loadTasks(tf.fs)
}

func (tf *TaskFiles) Get(id string) (task.TaskConfig, error) {

	tf.mu.RLock()
	defer tf.mu.RUnlock()

	tasks, err := loadTasks(tf.fs)
	if err != nil {
		return task.TaskConfig{}, err
	}

	t, ok := tasks[id]
	if !ok {
		return task.TaskConfig{}, fmt.Errorf("task %s: %w", id, types.ErrIsNotExist)
	}

	return t, nil
}

func (tf *TaskFiles) Save(t task.TaskConfig) error {

	tf.mu.Lock()
	defer tf.mu.Unlock()

	tasks, err := loadTasks(tf.fs)
	if err != nil {
		return err
	}
	tasks[t.Name] = t

	return flush(tf.fs, tasks)
}

func (tf *TaskFiles) Delete(id string) error {

	tf.mu.Lock()
	defer tf.mu.Unlock()

	tasks, err := loadTasks(tf.fs)
	if err != nil {
		return err
	}

	if _, ok := tasks[id]; !ok {
		return task.ErrIsNotExist
	}

	delete(tasks, id)

	return flush(tf.fs, tasks)
}

func loadTasks(fs *FileSystem) (map[string]task.TaskConfig, error) {
	data, err := fs.ReadFile(TaskConfigFile)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]task.TaskConfig{}, nil
		}
		return nil, err
	}
	if len(data) == 0 {
		return map[string]task.TaskConfig{}, nil
	}

	return UnmarshalTasks(data)
}

func flush(fs *FileSystem, tasks map[string]task.TaskConfig) error {
	data, err := MarshalTasks(tasks)
	if err != nil {
		return err
	}
	return fs.WriteToFile(TaskConfigFile, data)
}

type TaskDTO struct {
	Description string     `toml:"description"`
	Recipients  []agent.ID `toml:"recipients"`
	Reglament   string     `toml:"schedule"`
	Active      bool       `toml:"active"`
	Oneshot     bool       `toml:"once"`
	Request     string     `toml:"request" multiline:"true"`
}

func UnmarshalTasks(data []byte) (map[string]task.TaskConfig, error) {

	var dtos map[string]TaskDTO
	if err := toml.Unmarshal(data, &dtos); err != nil {
		return nil, err
	}

	tasks := make(map[string]task.TaskConfig, len(dtos))
	for k, v := range dtos {
		tasks[k] = task.TaskConfig{
			Name:        k,
			Description: v.Description,
			Recipients:  v.Recipients,
			Reglament:   v.Reglament,
			Active:      v.Active,
			Oneshot:     v.Oneshot,
			Request:     v.Request,
		}
	}

	return tasks, nil
}

func MarshalTasks(tasks map[string]task.TaskConfig) ([]byte, error) {

	dtos := map[string]TaskDTO{}

	for _, v := range tasks {
		dtos[v.Name] = TaskDTO{
			Description: v.Description,
			Recipients:  v.Recipients,
			Reglament:   v.Reglament,
			Active:      v.Active,
			Oneshot:     v.Oneshot,
			Request:     v.Request,
		}
	}

	rawData, err := toml.Marshal(dtos)
	if err != nil {
		return nil, err
	}

	return bytes.Join(
		[][]byte{[]byte(taskConfigDoc), rawData},
		[]byte("\n\n"),
	), nil
}

// tasksSent
func NewTasksReloader(tasksSvc *task.Service) sentinel.Action {
	return func(ctx context.Context, ev fsnotify.Event) error {
		return tasksSvc.Reload(ctx)
	}
}

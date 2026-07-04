package files

import (
	"arch-agent/internal/task"
	"arch-agent/internal/types"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sync"
)

const TaskFile = "tasks.json"

type TaskFiles struct {
	mu sync.RWMutex
	fs *FileSystem
}

func NewTaskFiles(fs *FileSystem) (*TaskFiles, error) {

	if _, err := fs.ReadFile(TaskFile); err != nil {
		if !errors.Is(err, types.ErrIsNotExist) {
			return nil, err
		}
		if err := fs.WriteToFile(TaskFile, []byte("[]")); err != nil {
			return nil, err
		}
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

func flush(fs *FileSystem, tasks map[string]task.TaskConfig) error {
	data, err := marshalTasks(tasks)
	if err != nil {
		return err
	}
	return fs.WriteToFile(TaskFile, data)
}

func unmarshalTasks(data []byte) (map[string]task.TaskConfig, error) {

	var dtos []task.TaskConfig
	if err := json.Unmarshal(data, &dtos); err != nil {
		return nil, err
	}

	tasks := make(map[string]task.TaskConfig, len(dtos))

	for _, dto := range dtos {
		tasks[dto.Name] = dto
	}

	return tasks, nil
}

func marshalTasks(tasks map[string]task.TaskConfig) ([]byte, error) {

	dtos := make([]task.TaskConfig, 0, len(tasks))
	for _, t := range tasks {
		dtos = append(dtos, t)
	}

	return json.MarshalIndent(dtos, "", "	")
}

func loadTasks(fs *FileSystem) (map[string]task.TaskConfig, error) {
	data, err := fs.ReadFile(TaskFile)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]task.TaskConfig{}, nil
		}
		return nil, err
	}
	if len(data) == 0 {
		return map[string]task.TaskConfig{}, nil
	}

	return unmarshalTasks(data)
}

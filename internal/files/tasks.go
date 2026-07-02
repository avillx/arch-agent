package files

import (
	"arch-agent/internal/agent"
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
	mu    sync.RWMutex
	tasks map[string]*task.TaskRecord
	fs    *FileSystem
}

func NewTaskFiles(fs *FileSystem) (*TaskFiles, error) {

	tasks, err := loadTasks(fs)
	if err != nil {
		return nil, err
	}

	return &TaskFiles{
		fs:    fs,
		tasks: tasks,
	}, nil
}

func (tf *TaskFiles) flush() error {
	data, err := marshalTasks(tf.tasks)
	if err != nil {
		return err
	}
	return tf.fs.WriteToFile(TaskFile, data)
}

func (tf *TaskFiles) All() (map[string]*task.TaskRecord, error) {
	tf.mu.RLock()
	defer tf.mu.RUnlock()
	return tf.tasks, nil
}

func (tf *TaskFiles) Get(id string) (*task.TaskRecord, error) {
	tf.mu.RLock()
	defer tf.mu.RUnlock()
	t, ok := tf.tasks[id]
	if !ok {
		return nil, types.ErrIsNotExist
	}
	return t, nil
}

func (tf *TaskFiles) Save(t *task.TaskRecord) error {
	tf.mu.Lock()
	defer tf.mu.Unlock()
	tf.tasks[t.Name()] = t
	return tf.flush()
}

func (tf *TaskFiles) Delete(id string) error {
	tf.mu.Lock()
	defer tf.mu.Unlock()
	delete(tf.tasks, id)
	return tf.flush()
}

type taskDTO struct {
	Active      bool       `json:"active"`
	Name        string     `json:"name"`
	Recipients  []agent.ID `json:"recipients"`
	Description string     `json:"description"`
	Request     string     `json:"request"`
	OneShot     bool       `json:"one_shot"`
	Reglament   string     `json:"reglament"`
}

func unmarshalTasks(data []byte) (map[string]*task.TaskRecord, error) {

	var dtos []taskDTO
	if err := json.Unmarshal(data, &dtos); err != nil {
		return nil, err
	}

	records := make(map[string]*task.TaskRecord, len(dtos))

	var errs []error
	for _, dto := range dtos {

		cfg, err := task.NewValidTaskConfig(
			dto.Name,
			dto.Description,
			dto.Recipients,
			dto.Reglament,
			dto.Request,
			dto.OneShot,
		)

		if err != nil {
			errs = append(errs, fmt.Errorf("can't load task %s: %w", dto.Name, err))
		}

		records[dto.Name] = &task.TaskRecord{
			Active:     dto.Active,
			TaskConfig: cfg,
		}
	}

	return records, errors.Join(errs...)
}

func marshalTasks(tasks map[string]*task.TaskRecord) ([]byte, error) {

	dtos := make([]taskDTO, 0, len(tasks))
	for _, t := range tasks {
		dto := taskDTO{
			Active:      t.Active,
			Name:        t.Name(),
			Description: t.Description(),
			Recipients:  t.Recipients(),
			Request:     t.Request(),
			OneShot:     t.Oneshot(),
			Reglament:   t.Reglament(),
		}
		dtos = append(dtos, dto)
	}

	return json.MarshalIndent(dtos, "", "	")
}

func loadTasks(fs *FileSystem) (map[string]*task.TaskRecord, error) {
	data, err := fs.ReadFile(TaskFile)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]*task.TaskRecord{}, nil
		}
		return nil, err
	}
	if len(data) == 0 {
		return map[string]*task.TaskRecord{}, nil
	}
	taskMap, err := unmarshalTasks(data)
	if err != nil {
		return nil, err
	}

	return taskMap, nil
}

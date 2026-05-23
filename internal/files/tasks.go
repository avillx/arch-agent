package files

import (
	"arch-agent/internal/agent"
	"arch-agent/internal/task"
	"arch-agent/internal/types"
	"encoding/json"
	"log/slog"
	"os"
	"sync"
)

const TaskFile = "tasks.json"

type TaskFiles struct {
	mu          sync.RWMutex
	tasks       map[string]*task.TaskRecord
	fs          *FileSystem
	cronFactory func(string) (task.Cron, error)
}

func NewTaskFiles(fs *FileSystem, cronFactory func(string) (task.Cron, error)) (*TaskFiles, error) {
	tf := &TaskFiles{
		fs:          fs,
		tasks:       map[string]*task.TaskRecord{},
		cronFactory: cronFactory,
	}
	return tf, tf.load()
}

func (tf *TaskFiles) load() error {
	data, err := tf.fs.ReadFile(TaskFile)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if len(data) == 0 {
		return nil
	}
	taskMap, err := unmarshalTasks(data, tf.cronFactory)
	if err != nil {
		return err
	}
	tf.tasks = taskMap
	return nil
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

func (tf *TaskFiles) Save(id string, t *task.TaskRecord) error {
	tf.mu.Lock()
	defer tf.mu.Unlock()
	tf.tasks[id] = t
	return tf.flush()
}

func (tf *TaskFiles) Delete(id string) error {
	tf.mu.Lock()
	defer tf.mu.Unlock()
	delete(tf.tasks, id)
	return tf.flush()
}

type reglamentDTO struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
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

func unmarshalTasks(data []byte, cronFactory func(string) (task.Cron, error)) (map[string]*task.TaskRecord, error) {

	var dtos []taskDTO
	if err := json.Unmarshal(data, &dtos); err != nil {
		return nil, err
	}

	records := make(map[string]*task.TaskRecord, len(dtos))
	for _, dto := range dtos {

		cron, err := cronFactory(dto.Reglament)
		if err != nil {
			slog.Error("unmarshal task, reglament parse", "task", dto.Name, "reglament", dto.Reglament, "error", err)
			continue
		}

		records[dto.Name] = &task.TaskRecord{
			Active: dto.Active,
			Task: task.NewTask(
				dto.Name,
				dto.Description,
				dto.Recipients,
				dto.Request,
				cron,
				dto.OneShot,
			),
		}
	}

	return records, nil
}

func marshalTasks(tasks map[string]*task.TaskRecord) ([]byte, error) {

	dtos := make([]taskDTO, 0, len(tasks))
	for _, t := range tasks {
		dto := taskDTO{
			Active:      t.Active,
			Name:        t.Name,
			Description: t.Description,
			Recipients:  t.Recipients,
			Request:     t.Request,
			OneShot:     t.OneShot,
			Reglament:   t.Reglament.Expression(),
		}
		dtos = append(dtos, dto)
	}

	return json.MarshalIndent(dtos, "", "	")
}

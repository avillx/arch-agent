package files

import (
	service "arch-agent/internal/app"
	"arch-agent/internal/domain/task"
	"arch-agent/internal/domain/types"
	"encoding/json"
	"fmt"
	"os"
	"sync"
)

const TaskFile = "tasks.json"

type TaskFiles struct {
	mu    sync.RWMutex
	tasks map[string]*service.TaskRecord
	fs    *FileSystem
}

func NewTaskFiles(fs *FileSystem) (*TaskFiles, error) {
	tf := &TaskFiles{
		fs:    fs,
		tasks: map[string]*service.TaskRecord{},
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
	taskMap, err := unmarshalTasks(data)
	if err != nil {
		return err
	}
	tf.tasks = taskMap
	return nil
}

type TaskRepo interface {
	All() (map[string]*service.TaskRecord, error)
	Get(id string) (*service.TaskRecord, error)
	Save(id string, t *service.TaskRecord) error
	Delete(id string) error
}

func (tf *TaskFiles) flush() error {
	data, err := marshalTasks(tf.tasks)
	if err != nil {
		return err
	}
	return tf.fs.WriteToFile(TaskFile, data)
}

func (tf *TaskFiles) All() (map[string]*service.TaskRecord, error) {
	tf.mu.RLock()
	defer tf.mu.RUnlock()
	return tf.tasks, nil
}

func (tf *TaskFiles) Get(id string) (*service.TaskRecord, error) {
	tf.mu.RLock()
	defer tf.mu.RUnlock()
	t, ok := tf.tasks[id]
	if !ok {
		return nil, types.ErrIsNotExist
	}
	return t, nil
}

func (tf *TaskFiles) Save(id string, t *service.TaskRecord) error {
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
	Active      bool             `json:"active"`
	Name        string           `json:"name"`
	Recipients  []task.Recipient `json:"recipients"`
	Description string           `json:"description"`
	Request     string           `json:"request"`
	OneShot     bool             `json:"one_shot"`
	Reglament   reglamentDTO     `json:"reglament"`
}

func marshalTasks(tasks map[string]*service.TaskRecord) ([]byte, error) {
	dtos := make(map[string]taskDTO, len(tasks))
	for id, t := range tasks {
		data, err := json.Marshal(t.Reglament)
		if err != nil {
			return nil, err
		}
		dtos[id] = taskDTO{
			Active: t.Active, Name: t.Name, Recipients: t.Recipients,
			Description: t.Description, Request: t.Request, OneShot: t.OneShot,
			Reglament: reglamentDTO{Type: t.Reglament.Type(), Data: data},
		}
	}
	return json.MarshalIndent(dtos, "", "	")
}

func unmarshalTasks(data []byte) (map[string]*service.TaskRecord, error) {
	var dtos map[string]taskDTO
	if err := json.Unmarshal(data, &dtos); err != nil {
		return nil, err
	}
	tasks := make(map[string]*service.TaskRecord, len(dtos))
	for id, dto := range dtos {
		r, err := unmarshalReglament(dto.Reglament.Type, dto.Reglament.Data)
		if err != nil {
			return nil, err
		}
		tasks[id] = &service.TaskRecord{
			Active: dto.Active,
			Task: task.Task{
				Name: dto.Name, Recipients: dto.Recipients,
				Description: dto.Description, Request: dto.Request, OneShot: dto.OneShot,
				Reglament: r,
			},
		}
	}
	return tasks, nil
}

func unmarshalReglament(typ string, data json.RawMessage) (task.Reglament, error) {
	switch typ {
	case "every":
		var r task.Every
		return r, json.Unmarshal(data, &r)
	case "daily":
		var r task.Daily
		return r, json.Unmarshal(data, &r)
	case "weekly":
		var r task.Weekly
		return r, json.Unmarshal(data, &r)
	case "monthly":
		var r task.Monthly
		return r, json.Unmarshal(data, &r)
	default:
		return nil, fmt.Errorf("unknown reglament type: %s", typ)
	}
}

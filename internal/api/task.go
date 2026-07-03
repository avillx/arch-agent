package api

import (
	"arch-agent/internal/agent"
	"arch-agent/internal/task"
	"arch-agent/internal/types"
	"errors"
	"net/http"
)

// handler
type TaskHandler struct {
	taskSvc *task.Service
}

// GET /task/all
func (s *TaskHandler) List(w http.ResponseWriter, _ *http.Request) error {

	tasks, err := s.taskSvc.All()
	if err != nil {
		return internal(err)
	}

	return respond(w, http.StatusOK, tasks)
}

// POST /task/{name}
func (s *TaskHandler) Create(w http.ResponseWriter, r *http.Request) error {

	type TaskConfigDTO struct {
		Name        string     `json:"name"`
		Description string     `json:"description"`
		Recipients  []agent.ID `json:"recipients"`
		Reglament   string     `json:"schedule"`
		Request     string     `json:"request"`
		Oneshot     bool       `json:"oneshot"`
	}

	dto, err := decode[TaskConfigDTO](r)
	if err != nil {
		return err
	}

	newTask, err := task.NewValidTaskConfig(
		dto.Name,
		dto.Description,
		dto.Recipients,
		dto.Reglament,
		dto.Request,
		dto.Oneshot,
	)
	if err != nil {
		var validationErr *types.ValidationError
		if errors.As(err, &validationErr) {
			return invalidRequest(validationErr.Problems())
		}
		return internal(err)
	}

	if err := s.taskSvc.AddTask(newTask); err != nil {
		return mapTaskServiceErr(err)
	}

	return respond(w, http.StatusOK, message("task created"))
}

// PATCH /task/{name}/start
func (s *TaskHandler) Start(w http.ResponseWriter, r *http.Request) error {
	taskName := r.PathValue("name")
	if err := s.taskSvc.Start(taskName); err != nil {
		return mapTaskServiceErr(err)
	}
	return respond(w, http.StatusOK, message("task started"))

}

// PATCH /task/{name}/stop
func (s *TaskHandler) Stop(w http.ResponseWriter, r *http.Request) error {
	taskName := r.PathValue("name")
	if err := s.taskSvc.Stop(taskName); err != nil {
		return mapTaskServiceErr(err)
	}
	return respond(w, http.StatusOK, message("task stopped"))
}

// PATCH /task/{name}
func (s *TaskHandler) Patch(w http.ResponseWriter, r *http.Request) error {
	taskName := r.PathValue("name")

	patch, err := decode[task.TaskPatch](r)
	if err != nil {
		return badRequest(err.Error())
	}

	if err := s.taskSvc.Patch(taskName, patch); err != nil {
		return mapTaskServiceErr(err)
	}

	return respond(w, http.StatusOK, message("task started"))
}

// DELETE /task/{name}
func (s *TaskHandler) Delete(w http.ResponseWriter, r *http.Request) error {
	taskName := r.PathValue("name")
	if err := s.taskSvc.Delete(taskName); err != nil {
		return mapTaskServiceErr(err)
	}
	return respond(w, http.StatusOK, message("task deleted"))
}

func mapTaskServiceErr(err error) error {
	switch {
	case errors.Is(err, task.ErrAlreadyRun):
		return badRequest(err.Error())
	case errors.Is(err, task.ErrTaskIsNotRunning):
		return badRequest(err.Error())
	case errors.Is(err, task.ErrCron):
		return badRequest(err.Error())
	case errors.Is(err, task.ErrAlreadyExist):
		return badRequest("task already exists")
	case errors.Is(err, task.ErrIsNotExist):
		return notFound("task does not exist")
	default:
		return internal(err)
	}
}

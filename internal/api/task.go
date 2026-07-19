package api

import (
	"arch-agent/internal/task"
	"arch-agent/internal/types"
	"errors"
	"net/http"
)

// handler
type taskHandler struct {
	taskSvc *task.Service
}

// GET /task/all
func (s *taskHandler) List(w http.ResponseWriter, _ *http.Request) error {

	type taskListDTO struct {
		Tasks []task.TaskConfig `json:"tasks"`
	}

	tasks, err := s.taskSvc.List()
	if err != nil {
		return internal(err)
	}

	dto := taskListDTO{
		Tasks: tasks,
	}

	return respond(w, http.StatusOK, dto)
}

// POST /task/{name}
func (s *taskHandler) Create(w http.ResponseWriter, r *http.Request) error {
	cfg, err := decode[task.TaskConfig](r)
	if err != nil {
		return err
	}

	if err := s.taskSvc.Add(cfg); err != nil {
		var validationErr *types.ValidationError
		if errors.As(err, &validationErr) {
			return invalidRequest(validationErr.Problems())
		}

		return mapTaskServiceErr(err)
	}

	return respond(w, http.StatusOK, message("task created"))
}

// PATCH /task/{name}
func (s *taskHandler) Patch(w http.ResponseWriter, r *http.Request) error {
	taskName := r.PathValue("name")

	patch, err := decode[task.TaskPatch](r)
	if err != nil {
		return badRequest(err.Error())
	}

	if err := s.taskSvc.Patch(taskName, patch); err != nil {
		return mapTaskServiceErr(err)
	}

	return respond(w, http.StatusOK, message("task patched"))
}

// DELETE /task/{name}
func (s *taskHandler) Delete(w http.ResponseWriter, r *http.Request) error {
	taskName := r.PathValue("name")
	if err := s.taskSvc.Delete(taskName); err != nil {
		return mapTaskServiceErr(err)
	}
	return respond(w, http.StatusOK, message("task deleted"))
}

func mapTaskServiceErr(err error) error {
	switch {
	case errors.Is(err, task.ErrCron):
		return badRequest(err.Error())
	case errors.Is(err, task.ErrNoRecipients):
		return badRequest(err.Error())
	case errors.Is(err, task.ErrAlreadyExist):
		return badRequest("task already exists")
	case errors.Is(err, task.ErrIsNotExist):
		return notFound("task does not exist")
	default:
		return internal(err)
	}
}

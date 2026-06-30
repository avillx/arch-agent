package api

import (
	"arch-agent/internal/agent"
	"arch-agent/internal/task"
	"arch-agent/internal/types"
	"context"
	"errors"
	"net/http"
	"regexp"
)

var cronRegex = regexp.MustCompile(`^(((?:[1-5]?[0-9])|(?:\*))(?:\/\d+)?(?:,(?:(?:[1-5]?[0-9])|(?:\*))(?:\/\d+)?)*)\s+(((?:[0-1]?[0-9]|2[0-3])|(?:\*))(?:\/\d+)?(?:,(?:(?:[0-1]?[0-9]|2[0-3])|(?:\*))(?:\/\d+)?)*)\s+(((?:[0-2]?[0-9]|3[0-1])|(?:\*))(?:\/\d+)?(?:,(?:(?:[0-2]?[0-9]|3[0-1])|(?:\*))(?:\/\d+)?)*)\s+(((?:[0-9]|1[0-2])|(?:\*))(?:\/\d+)?(?:,(?:(?:[0-9]|1[0-2])|(?:\*))(?:\/\d+)?)*)\s+(((?:[0-6])|(?:\*))(?:\/\d+)?(?:,(?:(?:[0-6])|(?:\*))(?:\/\d+)?)*)$`)

type TaskConfigDTO struct {
	Name        string     `json:"name"`
	Description string     `json:"description"`
	Recipients  []agent.ID `json:"recipients"`
	Reglament   string     `json:"schedule"`
	Request     string     `json:"request"`
	Oneshot     bool       `json:"oneshot"`
}

func (d *TaskConfigDTO) Validate(_ context.Context) map[string]string {
	problems := make(map[string]string)
	if d.Name == "" {
		problems["name"] = "must be not empty"
	}
	if d.Description == "" {
		problems["description"] = "must be not empty"
	}
	if !(len(d.Recipients) > 0) {
		problems["recipients"] = "must contain at least one recipient"
	}
	if !cronRegex.MatchString(d.Reglament) {
		problems["reglament"] = "bad format"
	}
	if d.Request == "" {
		problems["request"] = "must be not empty"
	}
	return problems
}

// handler
type TaskHandler struct {
	taskSvc *task.Service
}

// GET /task/all
func (s *TaskHandler) list(w http.ResponseWriter, _ *http.Request) error {

	tasks, err := s.taskSvc.All()
	if err != nil {
		return internal(err)
	}

	return respond(w, http.StatusOK, tasks)
}

// POST /task/{name}
func (s *TaskHandler) new(w http.ResponseWriter, r *http.Request) error {

	dto, err := decodeValid[*TaskConfigDTO](r)
	if err != nil {
		return err
	}

	if err := s.taskSvc.New(
		dto.Name,
		dto.Description,
		dto.Recipients,
		dto.Reglament,
		dto.Request,
		dto.Oneshot,
	); err != nil {
		return mapTaskServiceErr(err)
	}

	return respond(w, http.StatusOK, message("task created"))
}

// PATCH /task/{name}/start
func (s *TaskHandler) start(w http.ResponseWriter, r *http.Request) error {
	taskName := r.PathValue("name")
	if err := s.taskSvc.Start(taskName); err != nil {
		return mapTaskServiceErr(err)
	}
	return respond(w, http.StatusOK, message("task started"))

}

// PATCH /task/{name}/stop
func (s *TaskHandler) stop(w http.ResponseWriter, r *http.Request) error {
	taskName := r.PathValue("name")
	if err := s.taskSvc.Stop(taskName); err != nil {
		return mapTaskServiceErr(err)
	}
	return respond(w, http.StatusOK, message("task stopped"))
}

// DELETE /task/{name}
func (s *TaskHandler) delete(w http.ResponseWriter, r *http.Request) error {
	taskName := r.PathValue("name")
	if err := s.taskSvc.Delete(taskName); err != nil {
		return mapTaskServiceErr(err)
	}
	return respond(w, http.StatusOK, message("task deleted"))
}

func mapTaskServiceErr(err error) error {
	switch {
	// case errors.Is(err, task.ErrAlreadyExist):
	// 	return conflict("task already exists")
	case errors.Is(err, types.ErrIsNotExist):
		return notFound("task does not exist")
	default:
		return internal(err)
	}
}

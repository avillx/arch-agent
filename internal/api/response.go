package api

import (
	"arch-agent/internal/types"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
)

type apiError interface {
	error
	Status() int
	Body() any
}

type errStatus struct {
	status int
	msg    map[string]any
	err    error
}

func (e *errStatus) Error() string { return e.Error() }
func (e *errStatus) Unwrap() error { return e.err }

func internal(cause error) *errStatus {
	return &errStatus{
		status: http.StatusInternalServerError,
		msg:    map[string]any{"error": "internal error"},
		err:    cause,
	}
}

func badRequest(msg string) *errStatus {
	return &errStatus{
		status: http.StatusBadRequest,
		msg:    map[string]any{"error": msg},
	}
}

// validation failed
func invalidRequest(problems map[string]string) *errStatus {
	return &errStatus{
		status: http.StatusBadRequest,
		msg:    map[string]any{"problems": problems},
	}
}

func notFound(msg string) *errStatus {
	return &errStatus{
		status: http.StatusNotFound,
		msg:    map[string]any{"error": msg},
	}
}

func wrap(h func(w http.ResponseWriter, r *http.Request) error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := h(w, r)
		if err == nil {
			return
		}

		var es *errStatus
		if !errors.As(err, &es) {
			es = internal(err)
		}

		if es.err != nil {
			slog.Error("unhandled", "error", err)
		}

		if err := respond(w, es.status, es.msg); err != nil {
			slog.Error("response error", "error", err)
		}
	}
}

func decode[T any](r *http.Request) (T, error) {
	var v T
	if err := json.NewDecoder(r.Body).Decode(&v); err != nil {
		return v, badRequest(fmt.Sprintf("decode json: %s", err.Error()))
	}
	return v, nil
}

func decodeValid[T types.Validator](r *http.Request) (T, error) {
	var v T
	if err := json.NewDecoder(r.Body).Decode(&v); err != nil {
		return v, badRequest(fmt.Sprintf("decode json: %s", err.Error()))
	}
	if err := v.Validate(r.Context()); err != nil {

		if problems := types.ResovleValidationProblems(err); len(problems) > 0 {
			return v, invalidRequest(problems)
		}

		return v, err
	}

	return v, nil
}

func respond[T any](w http.ResponseWriter, status int, v T) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		return err
	}
	return nil
}

func message(content string) map[string]string {
	return map[string]string{
		"message": content,
	}
}

type Stream struct {
	w http.ResponseWriter
}

func newStream(w http.ResponseWriter) *Stream {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)
	return &Stream{w: w}
}

func (s *Stream) send(v any) {
	data, err := json.Marshal(v)
	if err != nil {
		return
	}
	_, _ = fmt.Fprintf(s.w, "data: %s\n\n", data)
	if f, ok := s.w.(http.Flusher); ok {
		f.Flush()
	}
}

func (s *Stream) done() {
	_, _ = fmt.Fprint(s.w, "data: [DONE]\n\n")
	if f, ok := s.w.(http.Flusher); ok {
		f.Flush()
	}
}

type ErrorDTO struct {
	Message string `json:"msg"`
}

// func (s *Stream) sendError(err error) {
// 	data, marshalErr := json.Marshal(ErrorDTO{Message: err.Error()})
// 	if marshalErr != nil {
// 		return
// 	}
// 	_, _ = fmt.Fprintf(s.w, "data: %s\n\n", data)
// 	if f, ok := s.w.(http.Flusher); ok {
// 		f.Flush()
// 	}
// }

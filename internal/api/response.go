package api

import (
	"arch-agent/internal/types"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
)

// Wrapper for simple status code response
type Response interface {
	StatusCode() int
}

type response struct {
	statusCode int
}

func NewResponse(code int) response {
	return response{
		statusCode: code,
	}
}

func (r response) StatusCode() int {
	return r.statusCode
}

// Wrapper for json responses
type JSONResponse interface {
	Response
	Content() map[string]any
}

type jsonResponse struct {
	response
	content any
}

func (r *jsonResponse) Content() any {
	return r.content
}

// content should be sended as response
// string wrapped as { "message" : "..." }
func NewJSONResponse[T string | any](code int, msg T) *jsonResponse {

	var s any
	s = msg

	if stringMessage, ok := any(msg).(string); ok {
		s = map[string]any{
			"message": stringMessage,
		}
	}

	return &jsonResponse{
		response: NewResponse(code),
		content:  s,
	}
}

type internalError struct {
	response
	cause error
}

// error shouldn't be sended. only logged. respond 500
func NewInternalError(cause error) Response {
	return &internalError{
		response: NewResponse(http.StatusInternalServerError),
		cause:    cause,
	}
}

// TODO: Err?
func (e *internalError) Error() string {
	return e.Error()
}

func NewBadRequest[T string | map[string]any](msg T) Response {
	return NewJSONResponse(http.StatusBadRequest, msg)
}

// send error as 400
// error message will be sended as respond
// unwrap validation error
// e.g. { "problems" : "..." }
func NewInvalidRequest(err error) Response {

	var r map[string]any

	if problems := types.ResovleValidationProblems(err); len(problems) > 0 {
		r = map[string]any{
			"problems": problems,
		}
	}

	if r == nil {
		r = map[string]any{
			"problems": err.Error(),
		}
	}

	return NewJSONResponse(http.StatusBadRequest, r)
}

func NewNotFound(cause string) Response {
	if cause != "" {
		return NewJSONResponse(http.StatusNotFound, cause)
	}
	return NewResponse(http.StatusNotFound)
}

// Responder
type APIResponder struct {
	*http.ServeMux
	logger *slog.Logger
}

func NewAPIResponder(logger *slog.Logger) *APIResponder {
	return &APIResponder{
		logger:   logger.WithGroup("api"),
		ServeMux: http.NewServeMux(),
	}
}

func (rs *APIResponder) HandleFunc(pattern string, handler func(http.ResponseWriter, *http.Request) Response) {
	rs.ServeMux.HandleFunc(pattern, rs.wrap(handler))
}

func (rs *APIResponder) wrap(h func(w http.ResponseWriter, r *http.Request) Response) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		response := h(w, r)

		rs.logger.Info("request", "status", response.StatusCode(), "path", r.URL.Path)

		// log error
		if err, ok := response.(error); ok {
			rs.logger.Error("internal", "error", err)
		}

		// respond json
		if jr, ok := response.(JSONResponse); ok {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(jr.StatusCode())
			enc := json.NewEncoder(w)
			if err := enc.Encode(jr.Content()); err != nil {
				rs.logger.Error("response encoding", "error", err)
			}
			return
		}

		// respond status code
		w.WriteHeader(response.StatusCode())
	}
}

func decode[T any](r *http.Request) (T, error) {
	var v T
	if err := json.NewDecoder(r.Body).Decode(&v); err != nil {
		return v, fmt.Errorf("broken json")
	}

	if validator, ok := any(v).(types.Validator); ok {
		if err := validator.Validate(r.Context()); err != nil {
			return v, err
		}
	}

	return v, nil
}

type Stream struct {
	w  http.ResponseWriter
	mu sync.Mutex
}

func newStream(w http.ResponseWriter) *Stream {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)
	return &Stream{w: w}
}

func (s *Stream) send(v any) {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := json.Marshal(v)
	if err != nil {
		return
	}
	fmt.Fprintf(s.w, "data: %s\n\n", data)
	if f, ok := s.w.(http.Flusher); ok {
		f.Flush()
	}
}

func (s *Stream) close() {
	s.mu.Lock()
	defer s.mu.Unlock()

	fmt.Fprint(s.w, "data: [DONE]\n\n")
	if f, ok := s.w.(http.Flusher); ok {
		f.Flush()
	}
}

type ErrorDTO struct {
	Message string `json:"msg"`
}

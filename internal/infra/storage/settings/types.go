package settingfiles

import (
	oaiadapter "arch-agent/internal/infra/openai"
	"reflect"
	"sync"
)

type Settings struct {
	// MCP           mcpadapter.MCPSettings `json:"mcp"`
	Reflection    oaiadapter.LLMSettings `json:"reflection"`
	Reasoning     oaiadapter.LLMSettings `json:"reasoning"`
	Summarization oaiadapter.LLMSettings `json:"summarization"`
	Dreaming      oaiadapter.LLMSettings `json:"dreaming"`
}

type RepoChangeNotifier[T any] struct {
	mu        sync.RWMutex
	value     T
	listeners []func(T)
}

func (r *RepoChangeNotifier[T]) Value() T {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.value
}

func (r *RepoChangeNotifier[T]) OnChange(f func(t T)) {

	r.listeners = append(r.listeners, f)
}

func (r *RepoChangeNotifier[T]) SetValue(v T) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if !reflect.DeepEqual(r.value, v) {
		r.value = v
		r.notify()
	}
}

func (r *RepoChangeNotifier[T]) notify() {
	for _, l := range r.listeners {
		l(r.value)
	}
}

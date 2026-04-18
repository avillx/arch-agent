package apiserver

import "net/http"

// Registers
func newRouter(wsclient any) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", handleHealth())
	// mux.HandleFunc("GET /tools", handleTools)
	// mux.HandleFunc("GET /memory", handleMemory)
	// mux.Handle("/ws", HANDLER_STRUCT_OR_INTERFACE)

	return mux
}

func handleHealth() http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}
}

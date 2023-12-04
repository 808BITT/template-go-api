package post

import (
	"net/http"
)

func Handler(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/api/health":
		health(w, r)
	default:
		http.Error(w, "Not found", http.StatusNotFound)
	}
}

func health(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Post OK"))
}

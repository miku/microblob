package microblob

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/thoas/stats"
)

// NewHandler sets up routes for serving and stats.
func NewHandler(backend Backend, blobfile string, loggingWriter io.Writer) http.Handler {
	metrics := stats.New()
	blobHandler := metrics.Handler(
		WithLastResponseTime(
			&BlobHandler{Backend: backend}))

	r := mux.NewRouter()
	r.Handle("/debug/vars", http.DefaultServeMux)
	r.HandleFunc("/stats", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(metrics.Data()); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	})
	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]interface{}{
			"name":    "microblob",
			"version": Version,
			"stats":   fmt.Sprintf("http://%s/stats", r.Host),
			"vars":    fmt.Sprintf("http://%s/debug/vars", r.Host),
		}); err != nil {
			http.Error(w, "could not serialize", http.StatusInternalServerError)
		}
	})
	r.HandleFunc("/count", func(w http.ResponseWriter, r *http.Request) {
		if c, ok := backend.(Counter); ok {
			if count, err := c.Count(); err == nil {
				if err := json.NewEncoder(w).Encode(map[string]interface{}{
					"count": count,
				}); err != nil {
					http.Error(w, "could not serialize", http.StatusInternalServerError)
				}
			}
		} else {
			http.Error(w, "not implemented", http.StatusNotFound)
		}
	})
	r.Handle("/update", UpdateHandler{Backend: backend, Blobfile: blobfile})
	r.Handle("/blob", blobHandler)     // Legacy route.
	r.Handle("/{key:.+}", blobHandler) // Preferred.

	return r
}

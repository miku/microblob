package microblob

import (
	"net/http"
	"strings"
)

// BlobHandler serves blobs.
type BlobHandler struct {
	Backend Backend
}

// ServeHTTP serves HTTP.
func (h *BlobHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	parts := filterEmpty(strings.Split(r.URL.Path, "/"))
	if len(parts) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`key is required`))
		return
	}
	key := strings.TrimSpace(parts[0])
	b, err := h.Backend.Get(key)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(err.Error()))
		return
	}
	w.Write(b)
}

// filterEmpty removes empty strings from a slice array.
func filterEmpty(ss []string) (filtered []string) {
	for _, s := range ss {
		if strings.TrimSpace(s) == "" {
			continue
		}
		filtered = append(filtered, s)
	}
	return
}

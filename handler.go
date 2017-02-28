package microblob

import (
	"expvar"
	"net/http"
	"strings"
)

var (
	okCounter  *expvar.Int
	errCounter *expvar.Int
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
		errCounter.Add(1)
		return
	}
	key := strings.TrimSpace(parts[0])
	b, err := h.Backend.Get(key)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(err.Error()))
		errCounter.Add(1)
		return
	}
	okCounter.Add(1)
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

func init() {
	okCounter = expvar.NewInt("okCounter")
	errCounter = expvar.NewInt("errCounter")
}

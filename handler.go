package microblob

import (
	"expvar"
	"net/http"
	"strings"
	"time"
)

var (
	okCounter          *expvar.Int
	legacyRouteCounter *expvar.Int
	errCounter         *expvar.Int
	lastResponseTime   *expvar.Float
)

// WithStats wraps a simple expvar benchmark around a handler.
func WithStats(h http.Handler) http.Handler {
	f := func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		h.ServeHTTP(w, r)
		lastResponseTime.Set(time.Since(started).Seconds())
	}
	return http.HandlerFunc(f)
}

// BlobHandler serves blobs.
type BlobHandler struct {
	Backend Backend
}

// ServeHTTP serves HTTP.
func (h *BlobHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var key string
	if r.URL.Path == "/blob" {
		// Legacy route. TODO(miku): Move to a saner route handling.
		key = strings.TrimSpace(r.URL.RawQuery)
		legacyRouteCounter.Add(1)
	} else {
		parts := filterEmpty(strings.Split(r.URL.Path, "/"))
		if len(parts) == 0 {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`key is required`))
			errCounter.Add(1)
			return
		}
		key = strings.TrimSpace(parts[0])
	}
	b, err := h.Backend.Get(key)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(err.Error()))
		errCounter.Add(1)
		return
	}
	w.Write(b)
	okCounter.Add(1)
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
	lastResponseTime = expvar.NewFloat("lastResponseTime")
	legacyRouteCounter = expvar.NewInt("legacyRouteCounter")
}

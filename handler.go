package microblob

import (
	"expvar"
	"net/http"
	"time"

	"github.com/gorilla/mux"
)

var (
	okCounter        *expvar.Int
	errCounter       *expvar.Int
	lastResponseTime *expvar.Float
)

// WithLastResponseTime keeps track of the last response time in exported variable
// lastResponseTime.
func WithLastResponseTime(h http.Handler) http.Handler {
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
	vars := mux.Vars(r)
	key, ok := vars["key"]
	if !ok {
		// From https://tools.ietf.org/html/rfc3986#section-3.4: [...] However, as query
		// components are often used to carry identifying information in the form of
		// "key=value" pairs [...]
		//
		// Legacy route with the key as value.
		key = r.URL.RawQuery
		if key == "" {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`key is required`))
			errCounter.Add(1)
			return
		}
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

func init() {
	okCounter = expvar.NewInt("okCounter")
	errCounter = expvar.NewInt("errCounter")
	lastResponseTime = expvar.NewFloat("lastResponseTime")
}

package microblob

import (
	"expvar"
	"io"
	"io/ioutil"
	"net/http"
	"os"
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
	w.Header().Set("X-Blob", Version)
	vars := mux.Vars(r)
	key, ok := vars["key"]
	if !ok || key == "blob/" {
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

// UpdateHandler adds more data to the blob server.
type UpdateHandler struct {
	Blobfile string
	Backend  Backend
}

// ServeHTTP appends data from POST body to existing blob file.
func (u UpdateHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	key := r.URL.Query().Get("key")
	if key == "" {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("update: key query parameter required"))
		return
	}
	extractor := ParsingExtractor{Key: key}
	f, err := ioutil.TempFile("", "microblob-")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}
	if _, err := io.Copy(f, r.Body); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("temporary copy failed: " + err.Error()))
		return
	}
	defer r.Body.Close()
	defer os.Remove(f.Name())
	if err := Append(u.Blobfile, f.Name(), u.Backend, extractor.ExtractKey); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("append: " + err.Error()))
		return
	}
	return
}

func init() {
	okCounter = expvar.NewInt("okCounter")
	errCounter = expvar.NewInt("errCounter")
	lastResponseTime = expvar.NewFloat("lastResponseTime")
}

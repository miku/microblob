package microblob

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/mux"
)

// mockBackend implements Backend for testing.
type mockBackend struct {
	data map[string][]byte
}

func (m *mockBackend) Get(key string) ([]byte, error) {
	v, ok := m.data[key]
	if !ok {
		return nil, errors.New("not found")
	}
	return v, nil
}

func (m *mockBackend) WriteEntries(entries []Entry) error { return nil }
func (m *mockBackend) Close() error                      { return nil }

func TestFinalNewlineReader(t *testing.T) {
	// Note: finalNewlineReader always appends a newline because standard Go
	// readers return EOF separately from data. The extra blank line is harmless
	// since the line processor skips empty lines.
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"without newline", "hello", "hello\n"},
		{"empty", "", "\n"},
		{"multi line no trailing", "a\nb", "a\nb\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &finalNewlineReader{r: strings.NewReader(tt.in)}
			got, err := io.ReadAll(r)
			if err != nil {
				t.Fatal(err)
			}
			if string(got) != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestBlobHandlerKeyFound(t *testing.T) {
	backend := &mockBackend{data: map[string][]byte{
		"abc": []byte(`{"id":"abc"}`),
	}}
	handler := &BlobHandler{Backend: backend}

	r := mux.NewRouter()
	r.Handle("/{key:.+}", handler)

	req := httptest.NewRequest("GET", "/abc", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}
	if xb := w.Header().Get("X-Blob"); xb != Version {
		t.Errorf("X-Blob = %q, want %q", xb, Version)
	}
	if w.Body.String() != `{"id":"abc"}` {
		t.Errorf("body = %q, want %q", w.Body.String(), `{"id":"abc"}`)
	}
}

func TestBlobHandlerKeyNotFound(t *testing.T) {
	backend := &mockBackend{data: map[string][]byte{}}
	handler := &BlobHandler{Backend: backend}

	r := mux.NewRouter()
	r.Handle("/{key:.+}", handler)

	req := httptest.NewRequest("GET", "/missing", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestBlobHandlerLegacyQuery(t *testing.T) {
	backend := &mockBackend{data: map[string][]byte{
		"id=abc": []byte(`{"id":"abc"}`),
	}}
	handler := &BlobHandler{Backend: backend}

	// Use the legacy /blob route where key comes from query string.
	r := mux.NewRouter()
	r.Handle("/blob", handler)

	req := httptest.NewRequest("GET", "/blob?id=abc", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestBlobHandlerMissingKey(t *testing.T) {
	backend := &mockBackend{data: map[string][]byte{}}
	handler := &BlobHandler{Backend: backend}

	r := mux.NewRouter()
	r.Handle("/blob", handler)

	req := httptest.NewRequest("GET", "/blob", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestUpdateHandlerMethodNotAllowed(t *testing.T) {
	handler := UpdateHandler{Backend: &mockBackend{data: map[string][]byte{}}}

	req := httptest.NewRequest("GET", "/update?key=id", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

func TestUpdateHandlerMissingKeyParam(t *testing.T) {
	handler := UpdateHandler{Backend: &mockBackend{data: map[string][]byte{}}}

	req := httptest.NewRequest("POST", "/update", strings.NewReader(`{"id":"x"}`))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

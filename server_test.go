package microblob

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/segmentio/encoding/json"
)

func TestNewHandlerRootEndpoint(t *testing.T) {
	backend := &mockBackend{data: map[string][]byte{}}
	handler := NewHandler(backend, "test.blob")

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}

	var info map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &info); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if info["name"] != "microblob" {
		t.Errorf("name = %v, want microblob", info["name"])
	}
	if info["version"] != Version {
		t.Errorf("version = %v, want %s", info["version"], Version)
	}
}

func TestNewHandlerCountEndpoint(t *testing.T) {
	dir := t.TempDir()
	blobPath := filepath.Join(dir, "test.blob")
	line := []byte(`{"id":"abc"}` + "\n")
	if err := os.WriteFile(blobPath, line, 0644); err != nil {
		t.Fatal(err)
	}

	backend := &LevelDBBackend{
		Filename: filepath.Join(dir, "test.db"),
		Blobfile: blobPath,
	}
	defer backend.Close()

	if err := backend.WriteEntries([]Entry{
		{Key: "abc", Offset: 0, Length: int64(len(line))},
	}); err != nil {
		t.Fatal(err)
	}

	handler := NewHandler(backend, blobPath)

	req := httptest.NewRequest("GET", "/count", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var result map[string]int64
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if result["count"] != 1 {
		t.Errorf("count = %d, want 1", result["count"])
	}
}

func TestNewHandlerCountNotImplemented(t *testing.T) {
	// mockBackend does not implement Counter, so /count should return 404.
	backend := &mockBackend{data: map[string][]byte{}}
	handler := NewHandler(backend, "test.blob")

	req := httptest.NewRequest("GET", "/count", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestNewHandlerBlobRoute(t *testing.T) {
	dir := t.TempDir()
	blobPath := filepath.Join(dir, "test.blob")
	line := []byte(`{"id":"xyz","v":1}` + "\n")
	if err := os.WriteFile(blobPath, line, 0644); err != nil {
		t.Fatal(err)
	}

	backend := &LevelDBBackend{
		Filename: filepath.Join(dir, "test.db"),
		Blobfile: blobPath,
	}
	defer backend.Close()

	if err := backend.WriteEntries([]Entry{
		{Key: "xyz", Offset: 0, Length: int64(len(line))},
	}); err != nil {
		t.Fatal(err)
	}

	handler := NewHandler(backend, blobPath)

	req := httptest.NewRequest("GET", "/xyz", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if w.Body.String() != string(line) {
		t.Errorf("body = %q, want %q", w.Body.String(), line)
	}
}

func TestNewHandlerStatsEndpoint(t *testing.T) {
	backend := &mockBackend{data: map[string][]byte{}}
	handler := NewHandler(backend, "test.blob")

	req := httptest.NewRequest("GET", "/stats", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}
}

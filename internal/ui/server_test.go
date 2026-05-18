package ui

import (
	"bytes"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bagadi-alnour/todo-cli/internal/storage"
	"github.com/bagadi-alnour/todo-cli/internal/types"
)

func TestServerCRUD(t *testing.T) {
	projectRoot := t.TempDir()
	if _, err := storage.InitProject(projectRoot, true); err != nil {
		t.Fatalf("init project: %v", err)
	}

	server := NewServer(projectRoot, 0)
	ln, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Skipf("skipping server tests: %v", err)
	}
	ts := httptest.NewUnstartedServer(server.Handler())
	ts.Listener = ln
	ts.Start()
	defer ts.Close()

	// Create
	createBody := `{"text":"first","paths":["src","README.md"],"priority":"high","tags":["api","backend"],"due":"2026-02-20"}`
	resp, err := http.Post(ts.URL+"/api/todos", "application/json", strings.NewReader(createBody))
	if err != nil {
		t.Fatalf("create todo request failed: %v", err)
	}
	defer resp.Body.Close()

	var createResp struct {
		Success bool       `json:"success"`
		Todo    types.Todo `json:"todo"`
		Error   string     `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&createResp); err != nil {
		t.Fatalf("decode create response: %v", err)
	}
	if !createResp.Success {
		t.Fatalf("create todo returned error: %s", createResp.Error)
	}

	todoID := createResp.Todo.ID
	if todoID == "" {
		t.Fatalf("expected todo id")
	}
	if createResp.Todo.Priority != types.PriorityHigh {
		t.Fatalf("expected priority high, got %s", createResp.Todo.Priority)
	}
	if len(createResp.Todo.Tags) != 2 {
		t.Fatalf("expected 2 tags, got %+v", createResp.Todo.Tags)
	}
	if createResp.Todo.DueAt == nil {
		t.Fatalf("expected due date to be set")
	}
	if got := createResp.Todo.Context.Paths; len(got) != 2 || got[0] != "src" || got[1] != "README.md" {
		t.Fatalf("expected paths [src README.md], got %+v", got)
	}

	// List
	resp, err = http.Get(ts.URL + "/api/todos")
	if err != nil {
		t.Fatalf("list todos request failed: %v", err)
	}
	defer resp.Body.Close()

	var listResp struct {
		Todos []types.Todo `json:"todos"`
		Count int          `json:"count"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&listResp); err != nil {
		t.Fatalf("decode list response: %v", err)
	}
	if listResp.Count != 1 || len(listResp.Todos) != 1 {
		t.Fatalf("expected 1 todo, got %+v", listResp)
	}

	// Update
	updatePayload := map[string]any{
		"status":   "blocked",
		"priority": "low",
		"paths":    []string{"docs", "internal/ui"},
		"tags":     []string{"ops"},
		"due":      "",
	}
	updateBytes, _ := json.Marshal(updatePayload)
	req, _ := http.NewRequest(http.MethodPut, ts.URL+"/api/todos/"+todoID, bytes.NewReader(updateBytes))
	req.Header.Set("Content-Type", "application/json")
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("update todo request failed: %v", err)
	}
	defer resp.Body.Close()

	var updateResp struct {
		Success bool       `json:"success"`
		Todo    types.Todo `json:"todo"`
		Error   string     `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&updateResp); err != nil {
		t.Fatalf("decode update response: %v", err)
	}
	if updateResp.Todo.Status != types.StatusBlocked || updateResp.Todo.Priority != types.PriorityLow {
		t.Fatalf("unexpected update result: %+v", updateResp.Todo)
	}
	if len(updateResp.Todo.Tags) != 1 || updateResp.Todo.Tags[0] != "ops" {
		t.Fatalf("expected tags [ops], got %+v", updateResp.Todo.Tags)
	}
	if updateResp.Todo.DueAt != nil {
		t.Fatalf("expected due date cleared, got %+v", updateResp.Todo.DueAt)
	}
	if got := updateResp.Todo.Context.Paths; len(got) != 2 || got[0] != "docs" || got[1] != "internal/ui" {
		t.Fatalf("expected updated paths [docs internal/ui], got %+v", got)
	}

	// Toggle
	resp, err = http.Post(ts.URL+"/api/todos/"+todoID+"/toggle", "application/json", nil)
	if err != nil {
		t.Fatalf("toggle todo request failed: %v", err)
	}
	resp.Body.Close()

	// Delete
	req, _ = http.NewRequest(http.MethodDelete, ts.URL+"/api/todos/"+todoID, nil)
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("delete todo request failed: %v", err)
	}
	resp.Body.Close()

	resp, err = http.Get(ts.URL + "/api/todos")
	if err != nil {
		t.Fatalf("list todos after delete failed: %v", err)
	}
	defer resp.Body.Close()
	if err := json.NewDecoder(resp.Body).Decode(&listResp); err != nil {
		t.Fatalf("decode list response after delete: %v", err)
	}
	if listResp.Count != 0 {
		t.Fatalf("expected 0 todos after delete, got %d", listResp.Count)
	}
}

func TestServerFiles(t *testing.T) {
	projectRoot := t.TempDir()
	if _, err := storage.InitProject(projectRoot, true); err != nil {
		t.Fatalf("init project: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(projectRoot, "src", "ui"), 0755); err != nil {
		t.Fatalf("create test dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(projectRoot, "README.md"), []byte("readme"), 0644); err != nil {
		t.Fatalf("create readme: %v", err)
	}
	if err := os.WriteFile(filepath.Join(projectRoot, "src", "app.go"), []byte("package src\n"), 0644); err != nil {
		t.Fatalf("create app: %v", err)
	}

	server := NewServer(projectRoot, 0)
	req := httptest.NewRequest(http.MethodGet, "/api/files", nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status OK, got %d: %s", rec.Code, rec.Body.String())
	}

	var rootResp struct {
		Dir     string `json:"dir"`
		Parent  string `json:"parent"`
		Entries []struct {
			Name string `json:"name"`
			Path string `json:"path"`
			Type string `json:"type"`
		} `json:"entries"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&rootResp); err != nil {
		t.Fatalf("decode files response: %v", err)
	}
	if rootResp.Dir != "" || rootResp.Parent != "" {
		t.Fatalf("unexpected root metadata: %+v", rootResp)
	}
	if len(rootResp.Entries) != 2 {
		t.Fatalf("expected visible root entries [src README.md], got %+v", rootResp.Entries)
	}
	if rootResp.Entries[0].Name != "src" || rootResp.Entries[0].Type != "dir" {
		t.Fatalf("expected src directory first, got %+v", rootResp.Entries)
	}
	if rootResp.Entries[1].Name != "README.md" || rootResp.Entries[1].Type != "file" {
		t.Fatalf("expected README.md file second, got %+v", rootResp.Entries)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/files?dir=src", nil)
	rec = httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status OK for src, got %d: %s", rec.Code, rec.Body.String())
	}
	var srcResp struct {
		Dir     string `json:"dir"`
		Parent  string `json:"parent"`
		Entries []struct {
			Name string `json:"name"`
			Path string `json:"path"`
			Type string `json:"type"`
		} `json:"entries"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&srcResp); err != nil {
		t.Fatalf("decode src files response: %v", err)
	}
	if srcResp.Dir != "src" || srcResp.Parent != "" {
		t.Fatalf("unexpected src metadata: %+v", srcResp)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/files?dir=..", nil)
	rec = httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status bad request for traversal, got %d", rec.Code)
	}
}

func TestServerPathsStayProjectRelative(t *testing.T) {
	projectRoot := t.TempDir()
	if _, err := storage.InitProject(projectRoot, true); err != nil {
		t.Fatalf("init project: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(projectRoot, "src"), 0755); err != nil {
		t.Fatalf("create src dir: %v", err)
	}
	projectFile := filepath.Join(projectRoot, "src", "app.go")
	if err := os.WriteFile(projectFile, []byte("package src\n"), 0644); err != nil {
		t.Fatalf("create app: %v", err)
	}

	server := NewServer(projectRoot, 0)
	createPayload := map[string]any{
		"text":  "relative path",
		"paths": []string{projectFile},
	}
	createBytes, _ := json.Marshal(createPayload)
	req := httptest.NewRequest(http.MethodPost, "/api/todos", bytes.NewReader(createBytes))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected create status OK, got %d: %s", rec.Code, rec.Body.String())
	}

	var createResp struct {
		Todo types.Todo `json:"todo"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&createResp); err != nil {
		t.Fatalf("decode create response: %v", err)
	}
	if got := createResp.Todo.Context.Paths; len(got) != 1 || got[0] != "src/app.go" {
		t.Fatalf("expected project-relative path src/app.go, got %+v", got)
	}

	outsideFile := filepath.Join(t.TempDir(), "outside.go")
	badPayload := map[string]any{
		"text":  "outside path",
		"paths": []string{outsideFile},
	}
	badBytes, _ := json.Marshal(badPayload)
	req = httptest.NewRequest(http.MethodPost, "/api/todos", bytes.NewReader(badBytes))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected bad request for outside absolute path, got %d", rec.Code)
	}

	traversalPayload := map[string]any{
		"text":  "traversal path",
		"paths": []string{"../outside.go"},
	}
	traversalBytes, _ := json.Marshal(traversalPayload)
	req = httptest.NewRequest(http.MethodPost, "/api/todos", bytes.NewReader(traversalBytes))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected bad request for path traversal, got %d", rec.Code)
	}
}

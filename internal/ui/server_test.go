package ui

import (
	"bytes"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
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
	createBody := `{"text":"first","path":"src","priority":"high"}`
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
		"path":     "docs",
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

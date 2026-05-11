package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/bagadi-alnour/todo-cli/internal/storage"
	"github.com/bagadi-alnour/todo-cli/internal/types"
)

func setupTestProject(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if _, err := storage.InitProject(dir, true); err != nil {
		t.Fatalf("init project: %v", err)
	}
	return dir
}

func chdir(t *testing.T, dir string) {
	t.Helper()
	orig, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() { os.Chdir(orig) })
}

func TestAddCommandJSON(t *testing.T) {
	dir := setupTestProject(t)
	chdir(t, dir)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"add", "Test task from integration", "--json", "--no-git", "--priority", "high"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("add command failed: %v", err)
	}

	var todo types.Todo
	if err := json.Unmarshal(buf.Bytes(), &todo); err != nil {
		t.Fatalf("failed to parse JSON output: %v\noutput: %s", err, buf.String())
	}
	if todo.Text != "Test task from integration" {
		t.Fatalf("expected text 'Test task from integration', got %q", todo.Text)
	}
	if todo.Priority != types.PriorityHigh {
		t.Fatalf("expected priority high, got %s", todo.Priority)
	}

	todos, err := storage.LoadTodos(dir)
	if err != nil {
		t.Fatalf("load todos: %v", err)
	}
	if len(todos) != 1 {
		t.Fatalf("expected 1 todo saved, got %d", len(todos))
	}
}

func TestListStaticJSON(t *testing.T) {
	dir := setupTestProject(t)
	chdir(t, dir)

	todos := []types.Todo{
		*types.NewTodo("id1", "first"),
		*types.NewTodo("id2", "second"),
	}
	todos[0].Priority = types.PriorityHigh
	todos[1].Status = types.StatusDone
	if err := storage.SaveTodos(dir, todos); err != nil {
		t.Fatalf("save: %v", err)
	}

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"list", "--json"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("list command failed: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("failed to parse JSON: %v\noutput: %s", err, buf.String())
	}
	count := int(result["count"].(float64))
	if count != 2 {
		t.Fatalf("expected count 2, got %d", count)
	}
}

func TestListFilterByStatus(t *testing.T) {
	dir := setupTestProject(t)
	chdir(t, dir)

	todos := []types.Todo{
		*types.NewTodo("id1", "open task"),
		*types.NewTodo("id2", "done task"),
	}
	todos[1].MarkDone()
	if err := storage.SaveTodos(dir, todos); err != nil {
		t.Fatalf("save: %v", err)
	}

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"list", "--json", "--status", "done"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("list command failed: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}
	count := int(result["count"].(float64))
	if count != 1 {
		t.Fatalf("expected 1 done todo, got %d", count)
	}
}

func TestEditCommand(t *testing.T) {
	dir := setupTestProject(t)
	chdir(t, dir)

	todos := []types.Todo{*types.NewTodo("abc123", "original text")}
	if err := storage.SaveTodos(dir, todos); err != nil {
		t.Fatalf("save: %v", err)
	}

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"edit", "1", "--text", "updated text", "--priority", "high"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("edit command failed: %v", err)
	}

	loaded, _ := storage.LoadTodos(dir)
	if loaded[0].Text != "updated text" {
		t.Fatalf("expected 'updated text', got %q", loaded[0].Text)
	}
	if loaded[0].Priority != types.PriorityHigh {
		t.Fatalf("expected priority high, got %s", loaded[0].Priority)
	}
}

func TestDoneCommand(t *testing.T) {
	dir := setupTestProject(t)
	chdir(t, dir)

	todos := []types.Todo{*types.NewTodo("abc123", "task to complete")}
	if err := storage.SaveTodos(dir, todos); err != nil {
		t.Fatalf("save: %v", err)
	}

	rootCmd.SetArgs([]string{"done", "1"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("done command failed: %v", err)
	}

	loaded, _ := storage.LoadTodos(dir)
	if loaded[0].Status != types.StatusDone {
		t.Fatalf("expected done, got %s", loaded[0].Status)
	}
}

func TestDeleteCommand(t *testing.T) {
	dir := setupTestProject(t)
	chdir(t, dir)

	todos := []types.Todo{
		*types.NewTodo("id1", "keep"),
		*types.NewTodo("id2", "delete me"),
	}
	if err := storage.SaveTodos(dir, todos); err != nil {
		t.Fatalf("save: %v", err)
	}

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"delete", "2"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("delete command failed: %v", err)
	}

	loaded, _ := storage.LoadTodos(dir)
	if len(loaded) != 1 {
		t.Fatalf("expected 1 todo after delete, got %d", len(loaded))
	}
	if loaded[0].ID != "id1" {
		t.Fatalf("wrong todo remained: %s", loaded[0].ID)
	}
}

func TestDoctorJSON(t *testing.T) {
	dir := setupTestProject(t)
	chdir(t, dir)

	todos := []types.Todo{
		*types.NewTodo("id1", "good task"),
		*types.NewTodo("id2", ""),
	}
	todos[0].Context.Paths = []string{"nonexistent.go"}
	if err := storage.SaveTodos(dir, todos); err != nil {
		t.Fatalf("save: %v", err)
	}

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"doctor", "--json"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("doctor command failed: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("failed to parse JSON: %v\noutput: %s", err, buf.String())
	}
	if result["healthy"].(bool) {
		t.Fatal("expected unhealthy report")
	}
}

func TestFindTodoByIDOrIndex_EdgeCases(t *testing.T) {
	todos := []types.Todo{
		{ID: "abcd1234efgh5678", Text: "first"},
		{ID: "abcd9999xxxx0000", Text: "second"},
		{ID: "zzzzaaaa11112222", Text: "third"},
	}

	// Partial ID match (unique prefix)
	if todo, _ := storage.FindTodoByIDOrIndex(todos, "abcd1234"); todo == nil || todo.Text != "first" {
		t.Fatal("partial ID match for 'abcd1234' failed")
	}

	// Partial ID match (shared prefix should match the first one)
	if todo, _ := storage.FindTodoByIDOrIndex(todos, "abcd"); todo == nil || todo.Text != "first" {
		t.Fatal("shared prefix match for 'abcd' should return first match")
	}

	// Index takes priority over ID
	if todo, _ := storage.FindTodoByIDOrIndex(todos, "1"); todo == nil || todo.Text != "first" {
		t.Fatal("index 1 should return first todo")
	}

	// Index out of range falls through to ID search
	if todo, _ := storage.FindTodoByIDOrIndex(todos, "99"); todo != nil {
		t.Fatal("out of range index should return nil")
	}

	// Very short partial (< 4 chars) should not partial-match
	if todo, _ := storage.FindTodoByIDOrIndex(todos, "abc"); todo != nil {
		t.Fatal("3-char partial should not match by ID prefix")
	}
}

func TestInitCommand(t *testing.T) {
	dir := t.TempDir()
	chdir(t, dir)

	rootCmd.SetArgs([]string{"init"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("init command failed: %v", err)
	}

	todosDir := filepath.Join(dir, ".todos")
	if _, err := os.Stat(todosDir); os.IsNotExist(err) {
		t.Fatal("expected .todos directory to be created")
	}
	if _, err := os.Stat(filepath.Join(todosDir, "todos.json")); os.IsNotExist(err) {
		t.Fatal("expected todos.json to be created")
	}
	if _, err := os.Stat(filepath.Join(todosDir, "config.json")); os.IsNotExist(err) {
		t.Fatal("expected config.json to be created")
	}
}

func TestShowCommandJSON(t *testing.T) {
	dir := setupTestProject(t)
	chdir(t, dir)

	todos := []types.Todo{*types.NewTodo("abc12345", "show me")}
	todos[0].Priority = types.PriorityHigh
	todos[0].Tags = []string{"backend"}
	if err := storage.SaveTodos(dir, todos); err != nil {
		t.Fatalf("save: %v", err)
	}

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"show", "1", "--json"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("show command failed: %v", err)
	}

	var todo types.Todo
	if err := json.Unmarshal(buf.Bytes(), &todo); err != nil {
		t.Fatalf("parse JSON: %v\noutput: %s", err, buf.String())
	}
	if todo.Text != "show me" {
		t.Fatalf("expected 'show me', got %q", todo.Text)
	}
	if todo.Priority != types.PriorityHigh {
		t.Fatalf("expected high, got %s", todo.Priority)
	}
}

func TestAddWithPathAndTag(t *testing.T) {
	dir := setupTestProject(t)
	chdir(t, dir)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"add", "Fix auth", "--path", "src/auth", "--tag", "backend", "--json", "--no-git"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("add failed: %v", err)
	}

	var todo types.Todo
	if err := json.Unmarshal(buf.Bytes(), &todo); err != nil {
		t.Fatalf("parse JSON: %v", err)
	}
	if len(todo.Context.Paths) != 1 || todo.Context.Paths[0] != "src/auth" {
		t.Fatalf("expected path src/auth, got %v", todo.Context.Paths)
	}
	if len(todo.Tags) != 1 || todo.Tags[0] != "backend" {
		t.Fatalf("expected tag backend, got %v", todo.Tags)
	}
}

func TestNextCommandJSON(t *testing.T) {
	dir := setupTestProject(t)
	chdir(t, dir)

	todos := []types.Todo{
		*types.NewTodo("lo1", "low prio"),
		*types.NewTodo("hi1", "high prio"),
	}
	todos[0].Priority = types.PriorityLow
	todos[1].Priority = types.PriorityHigh
	if err := storage.SaveTodos(dir, todos); err != nil {
		t.Fatalf("save: %v", err)
	}

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"next", "--json"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("next failed: %v", err)
	}

	var result map[string]json.RawMessage
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("parse JSON: %v\noutput: %s", err, buf.String())
	}
	var todo types.Todo
	if err := json.Unmarshal(result["todo"], &todo); err != nil {
		t.Fatalf("parse todo: %v", err)
	}
	if todo.Priority != types.PriorityHigh {
		t.Fatalf("expected high priority todo first, got %s", todo.Priority)
	}
}

func TestArchiveCommand(t *testing.T) {
	dir := setupTestProject(t)
	chdir(t, dir)

	todos := []types.Todo{
		*types.NewTodo("keep1", "still open"),
		*types.NewTodo("done1", "already done"),
	}
	todos[1].MarkDone()
	if err := storage.SaveTodos(dir, todos); err != nil {
		t.Fatalf("save: %v", err)
	}

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"archive", "--json"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("archive failed: %v", err)
	}

	var result map[string]json.RawMessage
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("parse JSON: %v\noutput: %s", err, buf.String())
	}

	remaining, _ := storage.LoadTodos(dir)
	if len(remaining) != 1 {
		t.Fatalf("expected 1 remaining todo, got %d", len(remaining))
	}
	if remaining[0].ID != "keep1" {
		t.Fatalf("wrong todo remained: %s", remaining[0].ID)
	}

	archived, _ := storage.LoadArchive(dir)
	if len(archived) != 1 {
		t.Fatalf("expected 1 archived todo, got %d", len(archived))
	}
}

func TestExportJSON(t *testing.T) {
	dir := setupTestProject(t)
	chdir(t, dir)

	todos := []types.Todo{*types.NewTodo("exp1", "exportable")}
	if err := storage.SaveTodos(dir, todos); err != nil {
		t.Fatalf("save: %v", err)
	}

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"export"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("export failed: %v", err)
	}

	var file types.TodoFile
	if err := json.Unmarshal(buf.Bytes(), &file); err != nil {
		t.Fatalf("parse JSON: %v", err)
	}
	if len(file.Todos) != 1 || file.Todos[0].Text != "exportable" {
		t.Fatalf("unexpected export content: %+v", file)
	}
}

func TestExportMarkdown(t *testing.T) {
	dir := setupTestProject(t)
	chdir(t, dir)

	todos := []types.Todo{*types.NewTodo("md1", "markdown task")}
	todos[0].Tags = []string{"docs"}
	if err := storage.SaveTodos(dir, todos); err != nil {
		t.Fatalf("save: %v", err)
	}

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"export", "--format", "markdown"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("export failed: %v", err)
	}

	out := buf.String()
	if !bytes.Contains([]byte(out), []byte("# Todos")) {
		t.Fatalf("expected markdown header, got: %s", out)
	}
	if !bytes.Contains([]byte(out), []byte("- [ ] markdown task")) {
		t.Fatalf("expected markdown task line, got: %s", out)
	}
}

func TestImportCommand(t *testing.T) {
	dir := setupTestProject(t)
	chdir(t, dir)

	importFile := filepath.Join(dir, "import.json")
	data, _ := json.Marshal(types.TodoFile{
		Version: 1,
		Todos: []types.Todo{
			*types.NewTodo("imp1", "imported task"),
		},
	})
	if err := os.WriteFile(importFile, data, 0644); err != nil {
		t.Fatalf("write import file: %v", err)
	}

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"import", importFile})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("import failed: %v", err)
	}

	loaded, _ := storage.LoadTodos(dir)
	if len(loaded) != 1 || loaded[0].Text != "imported task" {
		t.Fatalf("expected imported task, got %+v", loaded)
	}
}

func TestStatsCommandJSON(t *testing.T) {
	dir := setupTestProject(t)
	chdir(t, dir)

	todos := []types.Todo{
		*types.NewTodo("s1", "open task"),
		*types.NewTodo("s2", "done task"),
	}
	todos[1].MarkDone()
	if err := storage.SaveTodos(dir, todos); err != nil {
		t.Fatalf("save: %v", err)
	}

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"stats", "--json"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("stats failed: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("parse JSON: %v\noutput: %s", err, buf.String())
	}
	total := int(result["total"].(float64))
	if total != 2 {
		t.Fatalf("expected total 2, got %d", total)
	}
}

func TestSearchCommand(t *testing.T) {
	dir := setupTestProject(t)
	chdir(t, dir)

	todos := []types.Todo{
		*types.NewTodo("sr1", "Fix authentication bug"),
		*types.NewTodo("sr2", "Update docs"),
	}
	todos[0].Tags = []string{"auth"}
	if err := storage.SaveTodos(dir, todos); err != nil {
		t.Fatalf("save: %v", err)
	}

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"search", "auth", "--json"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("search failed: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("parse JSON: %v\noutput: %s", err, buf.String())
	}
	count := int(result["count"].(float64))
	if count != 1 {
		t.Fatalf("expected 1 match, got %d", count)
	}
}

func TestDoneByShortID(t *testing.T) {
	dir := setupTestProject(t)
	chdir(t, dir)

	todos := []types.Todo{*types.NewTodo("abcd1234efgh5678", "by-id task")}
	if err := storage.SaveTodos(dir, todos); err != nil {
		t.Fatalf("save: %v", err)
	}

	rootCmd.SetArgs([]string{"done", "abcd1234"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("done failed: %v", err)
	}

	loaded, _ := storage.LoadTodos(dir)
	if loaded[0].Status != types.StatusDone {
		t.Fatalf("expected done, got %s", loaded[0].Status)
	}
}

func TestDeleteByIndex(t *testing.T) {
	dir := setupTestProject(t)
	chdir(t, dir)

	todos := []types.Todo{
		*types.NewTodo("d1", "first"),
		*types.NewTodo("d2", "second"),
		*types.NewTodo("d3", "third"),
	}
	if err := storage.SaveTodos(dir, todos); err != nil {
		t.Fatalf("save: %v", err)
	}

	rootCmd.SetArgs([]string{"delete", "2"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("delete failed: %v", err)
	}

	loaded, _ := storage.LoadTodos(dir)
	if len(loaded) != 2 {
		t.Fatalf("expected 2 todos, got %d", len(loaded))
	}
	for _, td := range loaded {
		if td.ID == "d2" {
			t.Fatal("todo d2 should have been deleted")
		}
	}
}

func TestInvalidProjectError(t *testing.T) {
	dir := t.TempDir()
	chdir(t, dir)

	rootCmd.SetArgs([]string{"list", "--json"})
	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error for missing project")
	}
}

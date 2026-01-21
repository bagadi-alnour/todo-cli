package ui

import (
	"encoding/json"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/bagadi-alnour/todo-cli/internal/storage"
	"github.com/bagadi-alnour/todo-cli/internal/types"
)

// Server represents the web UI server
type Server struct {
	projectRoot string
	port        int
}

// NewServer creates a new UI server
func NewServer(projectRoot string, port int) *Server {
	return &Server{
		projectRoot: projectRoot,
		port:        port,
	}
}

// Handler returns the HTTP handler for the server
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()

	// Main page
	mux.HandleFunc("/", s.handleIndex)

	// API endpoints
	mux.HandleFunc("/api/todos", s.handleTodos)
	mux.HandleFunc("/api/todos/", s.handleTodoByID)
	mux.HandleFunc("/api/project", s.handleProject)

	return mux
}

// handleIndex serves the main HTML page
func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(indexHTML))
}

// handleTodos handles GET (list) and POST (create) for todos
func (s *Server) handleTodos(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == "OPTIONS" {
		return
	}

	switch r.Method {
	case "GET":
		s.listTodos(w, r)
	case "POST":
		s.createTodo(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleTodoByID handles operations on a single todo
func (s *Server) handleTodoByID(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, PUT, DELETE, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == "OPTIONS" {
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/api/todos/")
	parts := strings.Split(path, "/")

	if len(parts) == 0 || parts[0] == "" {
		http.NotFound(w, r)
		return
	}

	todoID := parts[0]

	// Check for /toggle endpoint
	if len(parts) == 2 && parts[1] == "toggle" && r.Method == "POST" {
		s.toggleTodo(w, r, todoID)
		return
	}

	switch r.Method {
	case "PUT":
		s.updateTodo(w, r, todoID)
	case "DELETE":
		s.deleteTodo(w, r, todoID)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleProject returns project information
func (s *Server) handleProject(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	projectName := filepath.Base(s.projectRoot)
	if projectName == "." || projectName == "" {
		projectName = "Project"
	}

	json.NewEncoder(w).Encode(map[string]string{
		"name": projectName,
		"path": s.projectRoot,
	})
}

// listTodos returns all todos
func (s *Server) listTodos(w http.ResponseWriter, r *http.Request) {
	todos, err := storage.LoadTodos(s.projectRoot)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"todos": todos,
		"count": len(todos),
	})
}

// createTodo creates a new todo
func (s *Server) createTodo(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Text     string  `json:"text"`
		Path     *string `json:"path"`
		Priority string  `json:"priority"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid request body"})
		return
	}

	if strings.TrimSpace(req.Text) == "" {
		json.NewEncoder(w).Encode(map[string]string{"error": "Todo text is required"})
		return
	}

	priority := types.Priority(strings.ToLower(req.Priority))
	if req.Priority != "" && !priority.IsValid() {
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid priority"})
		return
	}

	todos, err := storage.LoadTodos(s.projectRoot)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	id, err := storage.GenerateID()
	if err != nil {
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to generate ID"})
		return
	}

	todo := types.NewTodo(id, strings.TrimSpace(req.Text))
	if req.Path != nil && *req.Path != "" {
		todo.SetPaths([]string{*req.Path})
	}
	if req.Priority != "" && priority.IsValid() {
		todo.Priority = priority
	}

	todos = append(todos, *todo)

	if err := storage.SaveTodos(s.projectRoot, todos); err != nil {
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "todo": todo})
}

// toggleTodo toggles a todo's status
func (s *Server) toggleTodo(w http.ResponseWriter, r *http.Request, todoID string) {
	todos, err := storage.LoadTodos(s.projectRoot)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	todo, idx := storage.FindTodoByID(todos, todoID)
	if todo == nil {
		json.NewEncoder(w).Encode(map[string]string{"error": "Todo not found"})
		return
	}

	todos[idx].Toggle()

	if err := storage.SaveTodos(s.projectRoot, todos); err != nil {
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "todo": todos[idx]})
}

// updateTodo updates a todo
func (s *Server) updateTodo(w http.ResponseWriter, r *http.Request, todoID string) {
	var req struct {
		Text     string  `json:"text"`
		Status   string  `json:"status"`
		Path     *string `json:"path"`
		Priority string  `json:"priority"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid request body"})
		return
	}

	todos, err := storage.LoadTodos(s.projectRoot)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	todo, idx := storage.FindTodoByID(todos, todoID)
	if todo == nil {
		json.NewEncoder(w).Encode(map[string]string{"error": "Todo not found"})
		return
	}

	if req.Text != "" {
		todos[idx].Text = req.Text
	}
	if req.Status != "" {
		todos[idx].Status = types.Status(req.Status)
	}
	if req.Priority != "" {
		p := types.Priority(strings.ToLower(req.Priority))
		if !p.IsValid() {
			json.NewEncoder(w).Encode(map[string]string{"error": "Invalid priority"})
			return
		}
		todos[idx].Priority = p
	}
	if req.Path != nil {
		if *req.Path == "" {
			todos[idx].Context.Paths = []string{}
		} else {
			paths := strings.Split(*req.Path, ",")
			cleanPaths := []string{}
			for _, p := range paths {
				p = strings.TrimSpace(p)
				if p != "" {
					cleanPaths = append(cleanPaths, p)
				}
			}
			todos[idx].Context.Paths = cleanPaths
		}
	}
	todos[idx].UpdatedAt = time.Now()

	if err := storage.SaveTodos(s.projectRoot, todos); err != nil {
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "todo": todos[idx]})
}

// deleteTodo deletes a todo
func (s *Server) deleteTodo(w http.ResponseWriter, r *http.Request, todoID string) {
	todos, err := storage.LoadTodos(s.projectRoot)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	_, idx := storage.FindTodoByID(todos, todoID)
	if idx == -1 {
		json.NewEncoder(w).Encode(map[string]string{"error": "Todo not found"})
		return
	}

	todos = storage.DeleteTodo(todos, idx)

	if err := storage.SaveTodos(s.projectRoot, todos); err != nil {
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{"success": true})
}

// indexHTML is the embedded HTML template for the web UI
var indexHTML = `<!DOCTYPE html>
<html lang="en" data-theme="dark">
<head>
    <title>todo :: terminal</title>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <link rel="preconnect" href="https://fonts.googleapis.com">
    <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
    <link href="https://fonts.googleapis.com/css2?family=IBM+Plex+Mono:wght@400;500;600;700&family=Fira+Code:wght@400;500;600;700&display=swap" rel="stylesheet">
    <style>
        :root {
            /* Dark theme (terminal-inspired) */
            --bg-primary: #0a0a0a;
            --bg-secondary: #111111;
            --bg-tertiary: #1a1a1a;
            --bg-hover: #252525;
            --bg-input: #0d0d0d;
            --border-color: #2a2a2a;
            --border-focus: #00ff9f;
            --text-primary: #e0e0e0;
            --text-secondary: #808080;
            --text-muted: #4a4a4a;
            --accent-green: #00ff9f;
            --accent-cyan: #00d4ff;
            --accent-yellow: #ffcc00;
            --accent-red: #ff3366;
            --accent-purple: #bf7fff;
            --accent-orange: #ff9500;
            --accent-blue: #4d9fff;
            --glow-green: rgba(0, 255, 159, 0.15);
            --glow-cyan: rgba(0, 212, 255, 0.15);
            --shadow: 0 4px 20px rgba(0, 0, 0, 0.5);
            --radius: 4px;
            --scanline: repeating-linear-gradient(0deg, transparent, transparent 2px, rgba(0,0,0,0.03) 2px, rgba(0,0,0,0.03) 4px);
        }

        [data-theme="light"] {
            --bg-primary: #fafafa;
            --bg-secondary: #ffffff;
            --bg-tertiary: #f0f0f0;
            --bg-hover: #e8e8e8;
            --bg-input: #ffffff;
            --border-color: #d0d0d0;
            --border-focus: #00aa6f;
            --text-primary: #1a1a1a;
            --text-secondary: #666666;
            --text-muted: #999999;
            --accent-green: #00aa6f;
            --accent-cyan: #0099cc;
            --accent-yellow: #cc9900;
            --accent-red: #cc2244;
            --accent-purple: #8855cc;
            --accent-orange: #cc7700;
            --accent-blue: #3377cc;
            --glow-green: rgba(0, 170, 111, 0.1);
            --glow-cyan: rgba(0, 153, 204, 0.1);
            --shadow: 0 2px 10px rgba(0, 0, 0, 0.08);
            --scanline: none;
        }

        * { margin: 0; padding: 0; box-sizing: border-box; }

        body {
            font-family: 'IBM Plex Mono', 'Fira Code', monospace;
            background: var(--bg-primary);
            background-image: var(--scanline);
            color: var(--text-primary);
            min-height: 100vh;
            line-height: 1.6;
            font-size: 14px;
        }

        .app { max-width: 900px; margin: 0 auto; padding: 30px 20px; }

        /* Theme Toggle */
        .theme-toggle {
            position: fixed;
            top: 20px;
            right: 20px;
            width: 44px;
            height: 44px;
            background: var(--bg-tertiary);
            border: 1px solid var(--border-color);
            border-radius: var(--radius);
            cursor: pointer;
            display: flex;
            align-items: center;
            justify-content: center;
            transition: all 0.2s;
            z-index: 100;
            color: var(--text-secondary);
        }
        .theme-toggle:hover { border-color: var(--accent-green); color: var(--accent-green); }
        .theme-toggle svg { width: 20px; height: 20px; }

        /* Header */
        .header { margin-bottom: 30px; padding-bottom: 20px; border-bottom: 1px solid var(--border-color); }
        .header-row { display: flex; align-items: center; justify-content: space-between; flex-wrap: wrap; gap: 16px; }
        .header-left { display: flex; align-items: center; gap: 12px; }
        .terminal-icon { color: var(--accent-green); font-size: 1.5rem; }
        .header h1 {
            font-size: 1.3rem;
            font-weight: 600;
            color: var(--accent-green);
            letter-spacing: -0.5px;
        }
        .header h1 span { color: var(--text-muted); }
        .project-badge {
            display: inline-flex;
            align-items: center;
            gap: 8px;
            padding: 6px 12px;
            background: var(--bg-tertiary);
            border: 1px solid var(--border-color);
            border-radius: var(--radius);
            font-size: 0.8rem;
            color: var(--text-secondary);
        }
        .project-badge::before { content: "~/"; color: var(--accent-cyan); }

        /* Stats */
        .stats-row {
            display: flex;
            gap: 24px;
            margin-bottom: 24px;
            padding: 16px;
            background: var(--bg-secondary);
            border: 1px solid var(--border-color);
            border-radius: var(--radius);
            border-left: 3px solid var(--accent-green);
            flex-wrap: wrap;
        }
        .stat { display: flex; align-items: baseline; gap: 6px; }
        .stat-value { font-size: 1.4rem; font-weight: 700; }
        .stat-label { font-size: 0.75rem; text-transform: uppercase; color: var(--text-muted); letter-spacing: 1px; }
        .stat.total .stat-value { color: var(--text-primary); }
        .stat.open .stat-value { color: var(--accent-cyan); }
        .stat.done .stat-value { color: var(--accent-green); }
        .stat.blocked .stat-value { color: var(--accent-red); }
        .stat.waiting .stat-value { color: var(--accent-yellow); }
        .stat.tech-debt .stat-value { color: var(--accent-orange); }

        /* Add Form */
        .add-form {
            margin-bottom: 20px;
            padding: 16px;
            background: var(--bg-secondary);
            border: 1px solid var(--border-color);
            border-radius: var(--radius);
        }
        .add-form-label { display: flex; align-items: center; gap: 8px; margin-bottom: 12px; color: var(--accent-green); font-size: 0.8rem; font-weight: 500; }
        .add-form-label::before { content: "$"; color: var(--accent-cyan); }
        .add-form-row { display: flex; gap: 10px; }
        .add-input {
            flex: 1;
            background: var(--bg-input);
            border: 1px solid var(--border-color);
            border-radius: var(--radius);
            padding: 10px 14px;
            color: var(--text-primary);
            font-size: 0.9rem;
            font-family: inherit;
            transition: all 0.2s;
        }
        .add-input:focus { outline: none; border-color: var(--border-focus); box-shadow: 0 0 0 2px var(--glow-green); }
        .add-input::placeholder { color: var(--text-muted); }
        .path-input { max-width: 180px; }
        .add-btn {
            background: transparent;
            border: 1px solid var(--accent-green);
            border-radius: var(--radius);
            padding: 10px 20px;
            color: var(--accent-green);
            font-weight: 600;
            font-family: inherit;
            cursor: pointer;
            transition: all 0.2s;
            display: flex;
            align-items: center;
            gap: 6px;
        }
        .add-btn:hover { background: var(--accent-green); color: var(--bg-primary); }

        /* Filters */
        .filters { display: flex; gap: 6px; margin-bottom: 16px; flex-wrap: wrap; }
        .filter-btn {
            padding: 6px 14px;
            background: transparent;
            border: 1px solid var(--border-color);
            border-radius: var(--radius);
            color: var(--text-secondary);
            font-size: 0.8rem;
            font-weight: 500;
            font-family: inherit;
            cursor: pointer;
            transition: all 0.15s;
        }
        .filter-btn:hover { border-color: var(--text-secondary); color: var(--text-primary); }
        .filter-btn.active { background: var(--accent-green); border-color: var(--accent-green); color: var(--bg-primary); }
        .filter-select {
            padding: 6px 10px;
            background: var(--bg-secondary);
            border: 1px solid var(--border-color);
            border-radius: var(--radius);
            color: var(--text-secondary);
            font-size: 0.8rem;
            font-weight: 500;
            font-family: inherit;
            cursor: pointer;
        }
        .filter-select:focus { outline: none; border-color: var(--accent-green); }

        /* Todos Container */
        .todos-container {
            background: var(--bg-secondary);
            border: 1px solid var(--border-color);
            border-radius: var(--radius);
            overflow: hidden;
        }
        .todos-header {
            display: flex;
            padding: 10px 16px;
            background: var(--bg-tertiary);
            border-bottom: 1px solid var(--border-color);
            font-size: 0.75rem;
            text-transform: uppercase;
            letter-spacing: 1px;
            color: var(--text-muted);
            gap: 16px;
        }
        .todos-header span:first-child { width: 30px; }
        .todos-header span:nth-child(2) { flex: 1; }
        .todos-header span:nth-child(3) { width: 80px; }
        .todos-header span:last-child { width: 70px; }

        /* Todo Item */
        .todo-item {
            display: flex;
            align-items: flex-start;
            gap: 12px;
            padding: 14px 16px;
            border-bottom: 1px solid var(--border-color);
            transition: all 0.1s;
            position: relative;
        }
        .todo-item:last-child { border-bottom: none; }
        .todo-item:hover { background: var(--bg-hover); }
        .todo-item.selected { background: var(--glow-green); border-left: 2px solid var(--accent-green); padding-left: 14px; }

        .todo-index {
            width: 30px;
            font-size: 0.75rem;
            color: var(--text-muted);
            font-weight: 500;
            padding-top: 2px;
        }

        .todo-checkbox {
            width: 18px;
            height: 18px;
            border-radius: 3px;
            border: 2px solid var(--border-color);
            background: transparent;
            cursor: pointer;
            transition: all 0.15s;
            flex-shrink: 0;
            display: flex;
            align-items: center;
            justify-content: center;
            margin-top: 1px;
        }
        .todo-checkbox:hover { border-color: var(--accent-green); }
        .todo-item.done .todo-checkbox { background: var(--accent-green); border-color: var(--accent-green); }
        .todo-checkbox svg { width: 12px; height: 12px; opacity: 0; color: var(--bg-primary); stroke-width: 3; }
        .todo-item.done .todo-checkbox svg { opacity: 1; }

        .todo-content { flex: 1; min-width: 0; }
        .todo-text { font-size: 0.95rem; margin-bottom: 6px; word-wrap: break-word; line-height: 1.4; }
        .todo-item.done .todo-text { color: var(--text-muted); text-decoration: line-through; }

        .todo-meta { display: flex; align-items: center; gap: 10px; flex-wrap: wrap; font-size: 0.75rem; color: var(--text-muted); }
        .todo-status { padding: 2px 8px; border-radius: 3px; font-size: 0.65rem; font-weight: 600; text-transform: uppercase; letter-spacing: 0.5px; border: 1px solid; }
        .status-open { border-color: var(--accent-cyan); color: var(--accent-cyan); background: rgba(0, 212, 255, 0.08); }
        .status-done { border-color: var(--accent-green); color: var(--accent-green); background: rgba(0, 255, 159, 0.08); }
        .status-blocked { border-color: var(--accent-red); color: var(--accent-red); background: rgba(255, 51, 102, 0.08); }
        .status-waiting { border-color: var(--accent-yellow); color: var(--accent-yellow); background: rgba(255, 204, 0, 0.08); }
        .status-tech-debt { border-color: var(--accent-orange); color: var(--accent-orange); background: rgba(255, 149, 0, 0.08); }
        .todo-priority { padding: 2px 8px; border-radius: 3px; font-size: 0.65rem; font-weight: 700; letter-spacing: 0.5px; border: 1px solid; text-transform: uppercase; }
        .priority-high { border-color: var(--accent-red); color: var(--accent-red); background: rgba(255, 51, 102, 0.08); }
        .priority-medium { border-color: var(--accent-yellow); color: var(--accent-yellow); background: rgba(255, 204, 0, 0.08); }
        .priority-low { border-color: var(--accent-blue); color: var(--accent-blue); background: rgba(77, 159, 255, 0.08); }

        .todo-path { display: flex; align-items: center; gap: 4px; color: var(--accent-purple); }
        .todo-path::before { content: "📂"; font-size: 0.7rem; }
        .todo-branch { display: flex; align-items: center; gap: 4px; color: var(--accent-green); }
        .todo-branch::before { content: "⎇"; font-size: 0.8rem; }
        .todo-date { color: var(--text-muted); }

        .todo-actions { display: flex; gap: 4px; opacity: 0; transition: opacity 0.15s; }
        .todo-item:hover .todo-actions { opacity: 1; }

        .action-btn {
            width: 28px;
            height: 28px;
            border-radius: 3px;
            border: 1px solid transparent;
            background: transparent;
            color: var(--text-muted);
            cursor: pointer;
            display: flex;
            align-items: center;
            justify-content: center;
            transition: all 0.15s;
        }
        .action-btn:hover { background: var(--bg-tertiary); color: var(--text-primary); border-color: var(--border-color); }
        .action-btn.delete:hover { background: rgba(255, 51, 102, 0.1); border-color: var(--accent-red); color: var(--accent-red); }
        .action-btn svg { width: 14px; height: 14px; }

        /* Modal */
        .modal-overlay {
            position: fixed;
            inset: 0;
            background: rgba(0, 0, 0, 0.8);
            backdrop-filter: blur(4px);
            display: none;
            align-items: center;
            justify-content: center;
            z-index: 100;
        }
        .modal-overlay.active { display: flex; }
        .modal {
            background: var(--bg-secondary);
            border: 1px solid var(--border-color);
            border-radius: var(--radius);
            padding: 24px;
            width: 100%;
            max-width: 450px;
            margin: 20px;
            box-shadow: var(--shadow);
        }
        .modal h2 { font-size: 1rem; margin-bottom: 20px; display: flex; align-items: center; gap: 8px; color: var(--accent-green); font-weight: 600; }
        .modal h2::before { content: ">"; color: var(--accent-cyan); }
        .modal-field { margin-bottom: 14px; }
        .modal-field label { display: block; font-size: 0.75rem; color: var(--text-secondary); margin-bottom: 6px; text-transform: uppercase; letter-spacing: 0.5px; }
        .modal-field input, .modal-field select {
            width: 100%;
            background: var(--bg-input);
            border: 1px solid var(--border-color);
            border-radius: var(--radius);
            padding: 10px 12px;
            color: var(--text-primary);
            font-size: 0.9rem;
            font-family: inherit;
        }
        .modal-field input:focus, .modal-field select:focus { outline: none; border-color: var(--border-focus); }
        .modal-field select { cursor: pointer; }
        .modal-actions { display: flex; gap: 10px; justify-content: flex-end; margin-top: 20px; }
        .btn { padding: 8px 18px; border-radius: var(--radius); font-weight: 500; cursor: pointer; transition: all 0.15s; font-family: inherit; font-size: 0.85rem; }
        .btn-secondary { background: transparent; border: 1px solid var(--border-color); color: var(--text-secondary); }
        .btn-secondary:hover { border-color: var(--text-secondary); color: var(--text-primary); }
        .btn-primary { background: var(--accent-green); border: 1px solid var(--accent-green); color: var(--bg-primary); }
        .btn-primary:hover { filter: brightness(1.1); }
        .btn-danger { background: var(--accent-red); border: 1px solid var(--accent-red); color: white; }
        .btn-danger:hover { filter: brightness(1.1); }

        /* Empty State */
        .empty-state { text-align: center; padding: 50px 20px; color: var(--text-muted); }
        .empty-state .icon { font-size: 2.5rem; margin-bottom: 12px; opacity: 0.4; }
        .empty-state h3 { color: var(--text-secondary); margin-bottom: 6px; font-weight: 500; }
        .empty-state p { font-size: 0.85rem; }

        /* Shortcuts */
        .shortcuts {
            margin-top: 20px;
            padding: 14px 16px;
            background: var(--bg-secondary);
            border: 1px solid var(--border-color);
            border-radius: var(--radius);
        }
        .shortcuts-title { font-size: 0.75rem; color: var(--text-muted); margin-bottom: 10px; text-transform: uppercase; letter-spacing: 1px; }
        .shortcuts-grid { display: flex; flex-wrap: wrap; gap: 14px; }
        .shortcut { display: flex; align-items: center; gap: 6px; font-size: 0.8rem; color: var(--text-secondary); }
        kbd {
            background: var(--bg-tertiary);
            border: 1px solid var(--border-color);
            border-radius: 3px;
            padding: 2px 6px;
            font-family: inherit;
            font-size: 0.7rem;
            color: var(--accent-cyan);
            min-width: 22px;
            text-align: center;
        }

        /* Toast */
        .toast {
            position: fixed;
            bottom: 20px;
            right: 20px;
            background: var(--bg-tertiary);
            border: 1px solid var(--border-color);
            border-radius: var(--radius);
            padding: 10px 16px;
            display: flex;
            align-items: center;
            gap: 8px;
            box-shadow: var(--shadow);
            transform: translateY(100px);
            opacity: 0;
            transition: all 0.2s ease;
            z-index: 200;
            font-size: 0.85rem;
        }
        .toast.show { transform: translateY(0); opacity: 1; }
        .toast.success { border-left: 3px solid var(--accent-green); }
        .toast.error { border-left: 3px solid var(--accent-red); }

        /* Responsive */
        @media (max-width: 640px) {
            .app { padding: 16px; }
            .header h1 { font-size: 1.1rem; }
            .add-form-row { flex-direction: column; }
            .path-input { max-width: 100%; }
            .stats-row { gap: 16px; }
            .stat { flex-direction: column; gap: 2px; }
            .todo-actions { opacity: 1; }
            .todos-header { display: none; }
            .todo-index { display: none; }
            .theme-toggle { top: 10px; right: 10px; width: 38px; height: 38px; }
        }
    </style>
</head>
<body>
    <button class="theme-toggle" onclick="toggleTheme()" title="Toggle theme">
        <svg id="theme-icon-dark" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="5"/><line x1="12" y1="1" x2="12" y2="3"/><line x1="12" y1="21" x2="12" y2="23"/><line x1="4.22" y1="4.22" x2="5.64" y2="5.64"/><line x1="18.36" y1="18.36" x2="19.78" y2="19.78"/><line x1="1" y1="12" x2="3" y2="12"/><line x1="21" y1="12" x2="23" y2="12"/><line x1="4.22" y1="19.78" x2="5.64" y2="18.36"/><line x1="18.36" y1="5.64" x2="19.78" y2="4.22"/></svg>
        <svg id="theme-icon-light" style="display:none" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79z"/></svg>
    </button>

    <div class="app">
        <header class="header">
            <div class="header-row">
                <div class="header-left">
                    <span class="terminal-icon">▶</span>
                    <h1>todo<span>::cli</span></h1>
                </div>
                <div class="project-badge" id="project-name">loading...</div>
            </div>
        </header>

        <div class="stats-row" id="stats"></div>

        <div class="add-form">
            <div class="add-form-label">add_todo</div>
            <div class="add-form-row">
                <input type="text" class="add-input" id="new-todo-text" placeholder="What needs to be done?" autocomplete="off" />
                <input type="text" class="add-input path-input" id="new-todo-path" placeholder="path" autocomplete="off" />
                <select class="add-input path-input" id="new-todo-priority">
                    <option value="medium" selected>medium</option>
                    <option value="high">high</option>
                    <option value="low">low</option>
                </select>
                <button class="add-btn" onclick="addTodo()">+ add</button>
            </div>
        </div>

        <div class="filters">
            <button class="filter-btn active" data-filter="all">all</button>
            <button class="filter-btn" data-filter="open">open</button>
            <button class="filter-btn" data-filter="done">done</button>
            <button class="filter-btn" data-filter="blocked">blocked</button>
            <button class="filter-btn" data-filter="waiting">waiting</button>
            <button class="filter-btn" data-filter="tech-debt">debt</button>
            <select id="priority-filter" class="filter-select">
                <option value="all">priority: any</option>
                <option value="high">high first</option>
                <option value="medium">medium</option>
                <option value="low">low</option>
            </select>
        </div>

        <div class="todos-container">
            <div class="todos-header">
                <span>#</span>
                <span>task</span>
                <span>status</span>
                <span>actions</span>
            </div>
            <div id="todos"></div>
        </div>

        <div class="shortcuts">
            <div class="shortcuts-title">keybindings</div>
            <div class="shortcuts-grid">
                <div class="shortcut"><kbd>↑</kbd><kbd>↓</kbd> navigate</div>
                <div class="shortcut"><kbd>space</kbd> toggle</div>
                <div class="shortcut"><kbd>e</kbd> edit</div>
                <div class="shortcut"><kbd>d</kbd> delete</div>
                <div class="shortcut"><kbd>n</kbd> new</div>
                <div class="shortcut"><kbd>t</kbd> theme</div>
            </div>
        </div>
    </div>

    <div class="modal-overlay" id="edit-modal">
        <div class="modal">
            <h2>edit_todo</h2>
            <input type="hidden" id="edit-todo-id" />
            <div class="modal-field"><label>text</label><input type="text" id="edit-todo-text" /></div>
            <div class="modal-field"><label>status</label><select id="edit-todo-status"><option value="open">open</option><option value="done">done</option><option value="blocked">blocked</option><option value="waiting">waiting</option><option value="tech-debt">tech-debt</option></select></div>
            <div class="modal-field"><label>priority</label><select id="edit-todo-priority"><option value="high">high</option><option value="medium" selected>medium</option><option value="low">low</option></select></div>
            <div class="modal-field"><label>path</label><input type="text" id="edit-todo-path" placeholder="optional" /></div>
            <div class="modal-actions"><button class="btn btn-secondary" onclick="closeEditModal()">cancel</button><button class="btn btn-primary" onclick="saveEdit()">save</button></div>
        </div>
    </div>

    <div class="modal-overlay" id="delete-modal">
        <div class="modal">
            <h2>delete_todo</h2>
            <p style="color: var(--text-secondary); margin-bottom: 16px; font-size: 0.9rem;">This action cannot be undone.</p>
            <input type="hidden" id="delete-todo-id" />
            <div class="modal-actions"><button class="btn btn-secondary" onclick="closeDeleteModal()">cancel</button><button class="btn btn-danger" onclick="confirmDelete()">delete</button></div>
        </div>
    </div>

    <div class="toast" id="toast"><span id="toast-message"></span></div>

    <script>
        let currentFilter = 'all';
        let currentPriorityFilter = 'all';
        let allTodos = [];
        let selectedIndex = -1;
        let currentTheme = localStorage.getItem('todo-theme') || 'dark';

        document.addEventListener('DOMContentLoaded', () => {
            applyTheme(currentTheme);
            loadTodos();
            loadProjectInfo();
            setupEventListeners();
        });

        function toggleTheme() {
            currentTheme = currentTheme === 'dark' ? 'light' : 'dark';
            applyTheme(currentTheme);
            localStorage.setItem('todo-theme', currentTheme);
        }

        function applyTheme(theme) {
            document.documentElement.setAttribute('data-theme', theme);
            document.getElementById('theme-icon-dark').style.display = theme === 'dark' ? 'block' : 'none';
            document.getElementById('theme-icon-light').style.display = theme === 'light' ? 'block' : 'none';
        }

        function setupEventListeners() {
            document.querySelectorAll('.filter-btn').forEach(btn => {
                btn.addEventListener('click', () => {
                    currentFilter = btn.dataset.filter;
                    document.querySelectorAll('.filter-btn').forEach(b => b.classList.remove('active'));
                    btn.classList.add('active');
                    selectedIndex = -1;
                    renderTodos();
                });
            });
            document.getElementById('priority-filter').addEventListener('change', e => {
                currentPriorityFilter = e.target.value;
                selectedIndex = -1;
                renderTodos();
            });
            document.getElementById('new-todo-text').addEventListener('keypress', e => { if (e.key === 'Enter') addTodo(); });
            document.addEventListener('keydown', handleKeyboard);
            document.addEventListener('keydown', e => { if (e.key === 'Escape') { closeEditModal(); closeDeleteModal(); } });
            document.querySelectorAll('.modal-overlay').forEach(overlay => {
                overlay.addEventListener('click', e => { if (e.target === overlay) { closeEditModal(); closeDeleteModal(); } });
            });
        }

        async function loadProjectInfo() {
            try {
                const res = await fetch('/api/project');
                const data = await res.json();
                document.getElementById('project-name').textContent = data.name || 'project';
            } catch (err) { document.getElementById('project-name').textContent = 'project'; }
        }

        async function loadTodos() {
            try {
                const res = await fetch('/api/todos');
                const data = await res.json();
                allTodos = data.todos || [];
                renderStats();
                renderTodos();
            } catch (err) { showToast('Failed to load todos', 'error'); }
        }

        function renderStats() {
            const stats = [
                { key: 'total', label: 'total', value: allTodos.length },
                { key: 'open', label: 'open', value: allTodos.filter(t => t.status === 'open').length },
                { key: 'done', label: 'done', value: allTodos.filter(t => t.status === 'done').length },
                { key: 'blocked', label: 'blocked', value: allTodos.filter(t => t.status === 'blocked').length },
                { key: 'waiting', label: 'waiting', value: allTodos.filter(t => t.status === 'waiting').length },
                { key: 'tech-debt', label: 'debt', value: allTodos.filter(t => t.status === 'tech-debt').length }
            ];
            document.getElementById('stats').innerHTML = stats.map(s => '<div class="stat ' + s.key + '"><span class="stat-value">' + s.value + '</span><span class="stat-label">' + s.label + '</span></div>').join('');
        }

        function getFilteredTodos() {
            let filtered = allTodos.slice();
            if (currentFilter !== 'all') filtered = filtered.filter(t => t.status === currentFilter);
            if (currentPriorityFilter !== 'all') filtered = filtered.filter(t => normalizePriority(t.priority) === currentPriorityFilter);
            return sortByPriority(filtered);
        }

        function sortByPriority(todos) {
            return todos.slice().sort((a, b) => {
                const diff = priorityWeight(b.priority) - priorityWeight(a.priority);
                if (diff !== 0) return diff;
                return new Date(a.createdAt) - new Date(b.createdAt);
            });
        }

        function priorityMeta(priority) {
            const p = normalizePriority(priority);
            return { key: p, label: p === 'medium' ? 'med' : p };
        }

        function renderTodos() {
            const filtered = getFilteredTodos();
            const hasFilters = currentFilter !== 'all' || currentPriorityFilter !== 'all';
            if (filtered.length === 0) {
                document.getElementById('todos').innerHTML = '<div class="empty-state"><div class="icon">◇</div><h3>No todos</h3><p>' + (hasFilters ? 'Try a different filter' : 'Add your first todo above') + '</p></div>';
                return;
            }
            document.getElementById('todos').innerHTML = filtered.map((todo, i) => {
                const isDone = todo.status === 'done';
                const isSelected = i === selectedIndex;
                const paths = todo.context?.paths || [];
                const branch = todo.context?.branch || '';
                const priority = priorityMeta(todo.priority);
                return '<div class="todo-item' + (isDone ? ' done' : '') + (isSelected ? ' selected' : '') + '" data-id="' + todo.id + '" data-index="' + i + '">' +
                    '<span class="todo-index">' + String(i + 1).padStart(2, '0') + '</span>' +
                    '<div class="todo-checkbox" onclick="toggleTodo(\'' + todo.id + '\')"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor"><polyline points="20 6 9 17 4 12"/></svg></div>' +
                    '<div class="todo-content"><div class="todo-text">' + escapeHtml(todo.text) + '</div><div class="todo-meta">' +
                    '<span class="todo-status status-' + todo.status + '">' + todo.status + '</span>' +
                    '<span class="todo-priority priority-' + priority.key + '">' + priority.label + '</span>' +
                    '<span class="todo-date">' + formatDate(todo.createdAt) + '</span>' +
                    (paths.length > 0 ? '<span class="todo-path">' + escapeHtml(paths[0]) + '</span>' : '') +
                    (branch ? '<span class="todo-branch">' + escapeHtml(branch) + '</span>' : '') +
                    '</div></div>' +
                    '<div class="todo-actions">' +
                    '<button class="action-btn" onclick="openEditModal(\'' + todo.id + '\')" title="Edit"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"/><path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"/></svg></button>' +
                    '<button class="action-btn delete" onclick="openDeleteModal(\'' + todo.id + '\')" title="Delete"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="3 6 5 6 21 6"/><path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"/></svg></button>' +
                    '</div></div>';
            }).join('');
        }

        async function addTodo() {
            const text = document.getElementById('new-todo-text').value.trim();
            const path = document.getElementById('new-todo-path').value.trim();
            const priority = document.getElementById('new-todo-priority').value;
            if (!text) { showToast('Enter a todo', 'error'); return; }
            try {
                const res = await fetch('/api/todos', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ text, path: path || null, priority }) });
                if (res.ok) { document.getElementById('new-todo-text').value = ''; document.getElementById('new-todo-path').value = ''; document.getElementById('new-todo-priority').value = 'medium'; await loadTodos(); showToast('Added', 'success'); }
                else throw new Error('Failed');
            } catch (err) { showToast('Failed to add', 'error'); }
        }

        async function toggleTodo(id) { try { await fetch('/api/todos/' + id + '/toggle', { method: 'POST' }); await loadTodos(); } catch (err) { showToast('Toggle failed', 'error'); } }

        function openEditModal(id) {
            const todo = allTodos.find(t => t.id === id);
            if (!todo) return;
            document.getElementById('edit-todo-id').value = id;
            document.getElementById('edit-todo-text').value = todo.text;
            document.getElementById('edit-todo-status').value = todo.status;
            document.getElementById('edit-todo-priority').value = normalizePriority(todo.priority);
            document.getElementById('edit-todo-path').value = (todo.context?.paths || []).join(', ');
            document.getElementById('edit-modal').classList.add('active');
            setTimeout(() => document.getElementById('edit-todo-text').focus(), 100);
        }

        function closeEditModal() { document.getElementById('edit-modal').classList.remove('active'); }

        async function saveEdit() {
            const id = document.getElementById('edit-todo-id').value;
            const text = document.getElementById('edit-todo-text').value.trim();
            const status = document.getElementById('edit-todo-status').value;
            const priority = document.getElementById('edit-todo-priority').value;
            const path = document.getElementById('edit-todo-path').value.trim();
            if (!text) { showToast('Text required', 'error'); return; }
            try {
                const res = await fetch('/api/todos/' + id, { method: 'PUT', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ text, status, priority, path: path || null }) });
                if (res.ok) { closeEditModal(); await loadTodos(); showToast('Updated', 'success'); } else throw new Error('Failed');
            } catch (err) { showToast('Update failed', 'error'); }
        }

        function openDeleteModal(id) { document.getElementById('delete-todo-id').value = id; document.getElementById('delete-modal').classList.add('active'); }
        function closeDeleteModal() { document.getElementById('delete-modal').classList.remove('active'); }

        async function confirmDelete() {
            const id = document.getElementById('delete-todo-id').value;
            try {
                const res = await fetch('/api/todos/' + id, { method: 'DELETE' });
                if (res.ok) { closeDeleteModal(); await loadTodos(); showToast('Deleted', 'success'); } else throw new Error('Failed');
            } catch (err) { showToast('Delete failed', 'error'); }
        }

        function handleKeyboard(e) {
            const filtered = getFilteredTodos();
            const isModalOpen = document.querySelector('.modal-overlay.active');
            const isInputFocused = ['INPUT', 'TEXTAREA', 'SELECT'].includes(document.activeElement.tagName);
            if (isModalOpen || isInputFocused) return;
            switch (e.key) {
                case 'ArrowDown': case 'j': e.preventDefault(); selectedIndex = Math.min(selectedIndex + 1, filtered.length - 1); renderTodos(); scrollToSelected(); break;
                case 'ArrowUp': case 'k': e.preventDefault(); selectedIndex = Math.max(selectedIndex - 1, 0); renderTodos(); scrollToSelected(); break;
                case ' ': case 'Enter': e.preventDefault(); if (selectedIndex >= 0 && selectedIndex < filtered.length) toggleTodo(filtered[selectedIndex].id); break;
                case 'e': case 'E': if (selectedIndex >= 0 && selectedIndex < filtered.length) openEditModal(filtered[selectedIndex].id); break;
                case 'd': case 'D': if (selectedIndex >= 0 && selectedIndex < filtered.length) openDeleteModal(filtered[selectedIndex].id); break;
                case 'n': case 'N': document.getElementById('new-todo-text').focus(); break;
                case 't': case 'T': toggleTheme(); break;
            }
        }

        function scrollToSelected() { const selected = document.querySelector('.todo-item.selected'); if (selected) selected.scrollIntoView({ behavior: 'smooth', block: 'nearest' }); }
        function formatDate(dateStr) { const d = new Date(dateStr); return d.toLocaleDateString('en-US', { month: 'short', day: 'numeric' }); }
        function escapeHtml(text) { const div = document.createElement('div'); div.textContent = text; return div.innerHTML; }
        function normalizePriority(priority) { const p = (priority || 'medium').toString().toLowerCase(); return ['high', 'medium', 'low'].includes(p) ? p : 'medium'; }
        function priorityWeight(priority) { const p = normalizePriority(priority); if (p === 'high') return 3; if (p === 'low') return 1; return 2; }
        function showToast(message, type = 'success') { const toast = document.getElementById('toast'); toast.className = 'toast ' + type + ' show'; document.getElementById('toast-message').textContent = message; setTimeout(() => toast.classList.remove('show'), 2500); }
        setInterval(loadTodos, 10000);
    </script>
</body>
</html>`

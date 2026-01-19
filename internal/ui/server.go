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
		Text string  `json:"text"`
		Path *string `json:"path"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid request body"})
		return
	}

	if strings.TrimSpace(req.Text) == "" {
		json.NewEncoder(w).Encode(map[string]string{"error": "Todo text is required"})
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
		Text   string  `json:"text"`
		Status string  `json:"status"`
		Path   *string `json:"path"`
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
<html lang="en">
<head>
    <title>Todo System</title>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <link rel="preconnect" href="https://fonts.googleapis.com">
    <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
    <link href="https://fonts.googleapis.com/css2?family=JetBrains+Mono:wght@400;500;600;700&family=Inter:wght@400;500;600;700&display=swap" rel="stylesheet">
    <style>
        :root {
            --bg-primary: #0d1117;
            --bg-secondary: #161b22;
            --bg-tertiary: #21262d;
            --bg-hover: #30363d;
            --border-color: #30363d;
            --text-primary: #f0f6fc;
            --text-secondary: #8b949e;
            --text-muted: #6e7681;
            --accent-blue: #58a6ff;
            --accent-green: #3fb950;
            --accent-yellow: #d29922;
            --accent-red: #f85149;
            --accent-purple: #a371f7;
            --accent-cyan: #39c5cf;
            --accent-orange: #db6d28;
            --shadow: 0 8px 24px rgba(0,0,0,0.4);
            --radius: 12px;
            --radius-sm: 8px;
        }

        * { margin: 0; padding: 0; box-sizing: border-box; }

        body {
            font-family: 'Inter', -apple-system, BlinkMacSystemFont, sans-serif;
            background: var(--bg-primary);
            color: var(--text-primary);
            min-height: 100vh;
            line-height: 1.6;
        }

        .app { max-width: 1000px; margin: 0 auto; padding: 40px 20px; }

        .header { text-align: center; margin-bottom: 40px; }
        .logo { font-size: 3rem; margin-bottom: 8px; }
        .header h1 {
            font-size: 2rem;
            font-weight: 700;
            background: linear-gradient(135deg, var(--accent-cyan), var(--accent-purple));
            -webkit-background-clip: text;
            -webkit-text-fill-color: transparent;
            background-clip: text;
            margin-bottom: 8px;
        }
        .header .subtitle { color: var(--text-secondary); font-size: 1rem; }
        .project-badge {
            display: inline-flex;
            align-items: center;
            gap: 8px;
            margin-top: 16px;
            padding: 8px 16px;
            background: var(--bg-tertiary);
            border: 1px solid var(--border-color);
            border-radius: 20px;
            font-family: 'JetBrains Mono', monospace;
            font-size: 0.85rem;
            color: var(--accent-cyan);
        }

        .stats-grid {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(120px, 1fr));
            gap: 12px;
            margin-bottom: 32px;
        }

        .stat-card {
            background: var(--bg-secondary);
            border: 1px solid var(--border-color);
            border-radius: var(--radius);
            padding: 20px;
            text-align: center;
            transition: all 0.2s ease;
        }

        .stat-card:hover { border-color: var(--accent-cyan); transform: translateY(-2px); }
        .stat-number { font-size: 2rem; font-weight: 700; font-family: 'JetBrains Mono', monospace; }
        .stat-card.total .stat-number { color: var(--text-primary); }
        .stat-card.open .stat-number { color: var(--accent-blue); }
        .stat-card.done .stat-number { color: var(--accent-green); }
        .stat-card.blocked .stat-number { color: var(--accent-red); }
        .stat-card.waiting .stat-number { color: var(--accent-yellow); }
        .stat-card.tech-debt .stat-number { color: var(--accent-orange); }
        .stat-label { font-size: 0.75rem; text-transform: uppercase; letter-spacing: 1px; color: var(--text-secondary); margin-top: 4px; }

        .add-form {
            background: var(--bg-secondary);
            border: 1px solid var(--border-color);
            border-radius: var(--radius);
            padding: 20px;
            margin-bottom: 24px;
        }
        .add-form-header { display: flex; align-items: center; gap: 8px; margin-bottom: 16px; color: var(--text-secondary); font-size: 0.9rem; font-weight: 500; }
        .add-form-row { display: flex; gap: 12px; }
        .add-input {
            flex: 1;
            background: var(--bg-tertiary);
            border: 1px solid var(--border-color);
            border-radius: var(--radius-sm);
            padding: 12px 16px;
            color: var(--text-primary);
            font-size: 1rem;
            font-family: inherit;
            transition: all 0.2s;
        }
        .add-input:focus { outline: none; border-color: var(--accent-cyan); box-shadow: 0 0 0 3px rgba(57, 197, 207, 0.15); }
        .add-input::placeholder { color: var(--text-muted); }
        .path-input { width: 200px; flex: none; }
        .add-btn {
            background: linear-gradient(135deg, var(--accent-cyan), var(--accent-blue));
            border: none;
            border-radius: var(--radius-sm);
            padding: 12px 24px;
            color: white;
            font-weight: 600;
            cursor: pointer;
            transition: all 0.2s;
            display: flex;
            align-items: center;
            gap: 8px;
        }
        .add-btn:hover { transform: translateY(-1px); box-shadow: 0 4px 12px rgba(57, 197, 207, 0.3); }

        .filters { display: flex; gap: 8px; margin-bottom: 24px; flex-wrap: wrap; }
        .filter-btn {
            padding: 8px 16px;
            background: var(--bg-secondary);
            border: 1px solid var(--border-color);
            border-radius: 20px;
            color: var(--text-secondary);
            font-size: 0.85rem;
            font-weight: 500;
            cursor: pointer;
            transition: all 0.2s;
        }
        .filter-btn:hover { background: var(--bg-tertiary); color: var(--text-primary); }
        .filter-btn.active { background: var(--accent-cyan); border-color: var(--accent-cyan); color: var(--bg-primary); }

        .todos-container {
            background: var(--bg-secondary);
            border: 1px solid var(--border-color);
            border-radius: var(--radius);
            overflow: hidden;
        }

        .todo-item {
            display: flex;
            align-items: flex-start;
            gap: 16px;
            padding: 16px 20px;
            border-bottom: 1px solid var(--border-color);
            transition: all 0.15s;
            position: relative;
        }
        .todo-item:last-child { border-bottom: none; }
        .todo-item:hover { background: var(--bg-tertiary); }
        .todo-item.selected { background: rgba(57, 197, 207, 0.1); border-left: 3px solid var(--accent-cyan); }

        .todo-checkbox {
            width: 22px;
            height: 22px;
            border-radius: 6px;
            border: 2px solid var(--border-color);
            background: transparent;
            cursor: pointer;
            transition: all 0.2s;
            flex-shrink: 0;
            display: flex;
            align-items: center;
            justify-content: center;
            margin-top: 2px;
        }
        .todo-checkbox:hover { border-color: var(--accent-cyan); }
        .todo-item.done .todo-checkbox { background: var(--accent-green); border-color: var(--accent-green); }
        .todo-checkbox svg { width: 14px; height: 14px; opacity: 0; color: white; }
        .todo-item.done .todo-checkbox svg { opacity: 1; }

        .todo-content { flex: 1; min-width: 0; }
        .todo-text { font-size: 1rem; margin-bottom: 8px; word-wrap: break-word; }
        .todo-item.done .todo-text { color: var(--text-muted); text-decoration: line-through; }

        .todo-meta { display: flex; align-items: center; gap: 12px; flex-wrap: wrap; font-size: 0.8rem; color: var(--text-muted); }
        .todo-status { padding: 3px 8px; border-radius: 4px; font-size: 0.7rem; font-weight: 600; text-transform: uppercase; letter-spacing: 0.5px; }
        .status-open { background: rgba(88, 166, 255, 0.15); color: var(--accent-blue); }
        .status-done { background: rgba(63, 185, 80, 0.15); color: var(--accent-green); }
        .status-blocked { background: rgba(248, 81, 73, 0.15); color: var(--accent-red); }
        .status-waiting { background: rgba(210, 153, 34, 0.15); color: var(--accent-yellow); }
        .status-tech-debt { background: rgba(219, 109, 40, 0.15); color: var(--accent-orange); }

        .todo-path { display: flex; align-items: center; gap: 4px; font-family: 'JetBrains Mono', monospace; font-size: 0.75rem; color: var(--accent-purple); }
        .todo-branch { display: flex; align-items: center; gap: 4px; font-family: 'JetBrains Mono', monospace; font-size: 0.75rem; color: var(--accent-green); }

        .todo-actions { display: flex; gap: 8px; opacity: 0; transition: opacity 0.2s; }
        .todo-item:hover .todo-actions { opacity: 1; }

        .action-btn {
            width: 32px;
            height: 32px;
            border-radius: 6px;
            border: 1px solid var(--border-color);
            background: var(--bg-secondary);
            color: var(--text-secondary);
            cursor: pointer;
            display: flex;
            align-items: center;
            justify-content: center;
            transition: all 0.2s;
        }
        .action-btn:hover { background: var(--bg-hover); color: var(--text-primary); }
        .action-btn.delete:hover { background: rgba(248, 81, 73, 0.15); border-color: var(--accent-red); color: var(--accent-red); }
        .action-btn svg { width: 16px; height: 16px; }

        .modal-overlay {
            position: fixed;
            inset: 0;
            background: rgba(0, 0, 0, 0.7);
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
            max-width: 500px;
            margin: 20px;
            box-shadow: var(--shadow);
        }
        .modal h2 { font-size: 1.25rem; margin-bottom: 20px; display: flex; align-items: center; gap: 8px; }
        .modal-field { margin-bottom: 16px; }
        .modal-field label { display: block; font-size: 0.85rem; color: var(--text-secondary); margin-bottom: 8px; }
        .modal-field input, .modal-field select {
            width: 100%;
            background: var(--bg-tertiary);
            border: 1px solid var(--border-color);
            border-radius: var(--radius-sm);
            padding: 12px;
            color: var(--text-primary);
            font-size: 1rem;
            font-family: inherit;
        }
        .modal-field input:focus, .modal-field select:focus { outline: none; border-color: var(--accent-cyan); }
        .modal-actions { display: flex; gap: 12px; justify-content: flex-end; margin-top: 24px; }
        .btn { padding: 10px 20px; border-radius: var(--radius-sm); font-weight: 500; cursor: pointer; transition: all 0.2s; }
        .btn-secondary { background: var(--bg-tertiary); border: 1px solid var(--border-color); color: var(--text-primary); }
        .btn-secondary:hover { background: var(--bg-hover); }
        .btn-primary { background: var(--accent-cyan); border: none; color: var(--bg-primary); }
        .btn-primary:hover { box-shadow: 0 4px 12px rgba(57, 197, 207, 0.3); }
        .btn-danger { background: var(--accent-red); border: none; color: white; }
        .btn-danger:hover { box-shadow: 0 4px 12px rgba(248, 81, 73, 0.3); }

        .empty-state { text-align: center; padding: 60px 20px; color: var(--text-secondary); }
        .empty-state .icon { font-size: 4rem; margin-bottom: 16px; opacity: 0.5; }
        .empty-state h3 { color: var(--text-primary); margin-bottom: 8px; }

        .shortcuts {
            margin-top: 24px;
            padding: 16px 20px;
            background: var(--bg-secondary);
            border: 1px solid var(--border-color);
            border-radius: var(--radius);
        }
        .shortcuts-title { font-size: 0.85rem; color: var(--text-secondary); margin-bottom: 12px; display: flex; align-items: center; gap: 8px; }
        .shortcuts-grid { display: flex; flex-wrap: wrap; gap: 16px; }
        .shortcut { display: flex; align-items: center; gap: 8px; font-size: 0.85rem; }
        kbd {
            background: var(--bg-tertiary);
            border: 1px solid var(--border-color);
            border-radius: 4px;
            padding: 2px 8px;
            font-family: 'JetBrains Mono', monospace;
            font-size: 0.75rem;
            color: var(--accent-cyan);
        }

        .toast {
            position: fixed;
            bottom: 20px;
            right: 20px;
            background: var(--bg-tertiary);
            border: 1px solid var(--border-color);
            border-radius: var(--radius-sm);
            padding: 12px 20px;
            display: flex;
            align-items: center;
            gap: 10px;
            box-shadow: var(--shadow);
            transform: translateY(100px);
            opacity: 0;
            transition: all 0.3s ease;
            z-index: 200;
        }
        .toast.show { transform: translateY(0); opacity: 1; }
        .toast.success { border-left: 3px solid var(--accent-green); }
        .toast.error { border-left: 3px solid var(--accent-red); }

        @media (max-width: 640px) {
            .app { padding: 20px 16px; }
            .header h1 { font-size: 1.5rem; }
            .add-form-row { flex-direction: column; }
            .path-input { width: 100%; }
            .stats-grid { grid-template-columns: repeat(3, 1fr); }
            .todo-actions { opacity: 1; }
        }
    </style>
</head>
<body>
    <div class="app">
        <header class="header">
            <div class="logo">📋</div>
            <h1>Todo System</h1>
            <p class="subtitle">Todos that understand your code</p>
            <div class="project-badge"><span>📁</span><span id="project-name">Loading...</span></div>
        </header>

        <div class="stats-grid" id="stats"></div>

        <div class="add-form">
            <div class="add-form-header"><span>✨</span><span>Add new todo</span></div>
            <div class="add-form-row">
                <input type="text" class="add-input" id="new-todo-text" placeholder="What needs to be done?" />
                <input type="text" class="add-input path-input" id="new-todo-path" placeholder="Path (optional)" />
                <button class="add-btn" onclick="addTodo()">
                    <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><line x1="12" y1="5" x2="12" y2="19"/><line x1="5" y1="12" x2="19" y2="12"/></svg>
                    Add
                </button>
            </div>
        </div>

        <div class="filters">
            <button class="filter-btn active" data-filter="all">All</button>
            <button class="filter-btn" data-filter="open">Open</button>
            <button class="filter-btn" data-filter="done">Done</button>
            <button class="filter-btn" data-filter="blocked">Blocked</button>
            <button class="filter-btn" data-filter="waiting">Waiting</button>
            <button class="filter-btn" data-filter="tech-debt">Tech Debt</button>
        </div>

        <div class="todos-container"><div id="todos"></div></div>

        <div class="shortcuts">
            <div class="shortcuts-title"><span>⌨️</span><span>Keyboard shortcuts</span></div>
            <div class="shortcuts-grid">
                <div class="shortcut"><kbd>↑</kbd><kbd>↓</kbd> Navigate</div>
                <div class="shortcut"><kbd>Space</kbd> Toggle</div>
                <div class="shortcut"><kbd>E</kbd> Edit</div>
                <div class="shortcut"><kbd>D</kbd> Delete</div>
                <div class="shortcut"><kbd>N</kbd> New todo</div>
            </div>
        </div>
    </div>

    <div class="modal-overlay" id="edit-modal">
        <div class="modal">
            <h2>✏️ Edit Todo</h2>
            <input type="hidden" id="edit-todo-id" />
            <div class="modal-field"><label>Todo text</label><input type="text" id="edit-todo-text" /></div>
            <div class="modal-field"><label>Status</label><select id="edit-todo-status"><option value="open">Open</option><option value="done">Done</option><option value="blocked">Blocked</option><option value="waiting">Waiting</option><option value="tech-debt">Tech Debt</option></select></div>
            <div class="modal-field"><label>Path (optional)</label><input type="text" id="edit-todo-path" /></div>
            <div class="modal-actions"><button class="btn btn-secondary" onclick="closeEditModal()">Cancel</button><button class="btn btn-primary" onclick="saveEdit()">Save Changes</button></div>
        </div>
    </div>

    <div class="modal-overlay" id="delete-modal">
        <div class="modal">
            <h2>🗑️ Delete Todo</h2>
            <p style="color: var(--text-secondary); margin-bottom: 20px;">Are you sure you want to delete this todo? This action cannot be undone.</p>
            <input type="hidden" id="delete-todo-id" />
            <div class="modal-actions"><button class="btn btn-secondary" onclick="closeDeleteModal()">Cancel</button><button class="btn btn-danger" onclick="confirmDelete()">Delete</button></div>
        </div>
    </div>

    <div class="toast" id="toast"><span id="toast-message"></span></div>

    <script>
        let currentFilter = 'all';
        let allTodos = [];
        let selectedIndex = -1;

        document.addEventListener('DOMContentLoaded', () => {
            loadTodos();
            loadProjectInfo();
            setupEventListeners();
        });

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
                document.getElementById('project-name').textContent = data.name || 'Project';
            } catch (err) { document.getElementById('project-name').textContent = 'Project'; }
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
                { key: 'total', label: 'Total', value: allTodos.length },
                { key: 'open', label: 'Open', value: allTodos.filter(t => t.status === 'open').length },
                { key: 'done', label: 'Done', value: allTodos.filter(t => t.status === 'done').length },
                { key: 'blocked', label: 'Blocked', value: allTodos.filter(t => t.status === 'blocked').length },
                { key: 'waiting', label: 'Waiting', value: allTodos.filter(t => t.status === 'waiting').length },
                { key: 'tech-debt', label: 'Tech Debt', value: allTodos.filter(t => t.status === 'tech-debt').length }
            ];
            document.getElementById('stats').innerHTML = stats.map(s => '<div class="stat-card ' + s.key + '"><div class="stat-number">' + s.value + '</div><div class="stat-label">' + s.label + '</div></div>').join('');
        }

        function renderTodos() {
            let filtered = allTodos;
            if (currentFilter !== 'all') filtered = allTodos.filter(t => t.status === currentFilter);
            if (filtered.length === 0) {
                document.getElementById('todos').innerHTML = '<div class="empty-state"><div class="icon">📝</div><h3>No todos found</h3><p>' + (currentFilter === 'all' ? 'Add your first todo above!' : 'Try a different filter') + '</p></div>';
                return;
            }
            document.getElementById('todos').innerHTML = filtered.map((todo, i) => {
                const isDone = todo.status === 'done';
                const isSelected = i === selectedIndex;
                const paths = todo.context?.paths || [];
                const branch = todo.context?.branch || '';
                return '<div class="todo-item' + (isDone ? ' done' : '') + (isSelected ? ' selected' : '') + '" data-id="' + todo.id + '" data-index="' + i + '">' +
                    '<div class="todo-checkbox" onclick="toggleTodo(\'' + todo.id + '\')"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3"><polyline points="20 6 9 17 4 12"/></svg></div>' +
                    '<div class="todo-content"><div class="todo-text">' + escapeHtml(todo.text) + '</div><div class="todo-meta">' +
                    '<span class="todo-status status-' + todo.status + '">' + todo.status + '</span>' +
                    '<span>' + formatDate(todo.createdAt) + '</span>' +
                    (paths.length > 0 ? '<span class="todo-path">📁 ' + escapeHtml(paths.join(', ')) + '</span>' : '') +
                    (branch ? '<span class="todo-branch">🌿 ' + escapeHtml(branch) + '</span>' : '') +
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
            if (!text) { showToast('Please enter a todo', 'error'); return; }
            try {
                const res = await fetch('/api/todos', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ text, path: path || null }) });
                if (res.ok) { document.getElementById('new-todo-text').value = ''; document.getElementById('new-todo-path').value = ''; await loadTodos(); showToast('Todo added!', 'success'); }
                else throw new Error('Failed');
            } catch (err) { showToast('Failed to add todo', 'error'); }
        }

        async function toggleTodo(id) { try { await fetch('/api/todos/' + id + '/toggle', { method: 'POST' }); await loadTodos(); } catch (err) { showToast('Failed to toggle todo', 'error'); } }

        function openEditModal(id) {
            const todo = allTodos.find(t => t.id === id);
            if (!todo) return;
            document.getElementById('edit-todo-id').value = id;
            document.getElementById('edit-todo-text').value = todo.text;
            document.getElementById('edit-todo-status').value = todo.status;
            document.getElementById('edit-todo-path').value = (todo.context?.paths || []).join(', ');
            document.getElementById('edit-modal').classList.add('active');
        }

        function closeEditModal() { document.getElementById('edit-modal').classList.remove('active'); }

        async function saveEdit() {
            const id = document.getElementById('edit-todo-id').value;
            const text = document.getElementById('edit-todo-text').value.trim();
            const status = document.getElementById('edit-todo-status').value;
            const path = document.getElementById('edit-todo-path').value.trim();
            if (!text) { showToast('Todo text cannot be empty', 'error'); return; }
            try {
                const res = await fetch('/api/todos/' + id, { method: 'PUT', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ text, status, path: path || null }) });
                if (res.ok) { closeEditModal(); await loadTodos(); showToast('Todo updated!', 'success'); } else throw new Error('Failed');
            } catch (err) { showToast('Failed to update todo', 'error'); }
        }

        function openDeleteModal(id) { document.getElementById('delete-todo-id').value = id; document.getElementById('delete-modal').classList.add('active'); }
        function closeDeleteModal() { document.getElementById('delete-modal').classList.remove('active'); }

        async function confirmDelete() {
            const id = document.getElementById('delete-todo-id').value;
            try {
                const res = await fetch('/api/todos/' + id, { method: 'DELETE' });
                if (res.ok) { closeDeleteModal(); await loadTodos(); showToast('Todo deleted!', 'success'); } else throw new Error('Failed');
            } catch (err) { showToast('Failed to delete todo', 'error'); }
        }

        function handleKeyboard(e) {
            const filtered = currentFilter === 'all' ? allTodos : allTodos.filter(t => t.status === currentFilter);
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
            }
        }

        function scrollToSelected() { const selected = document.querySelector('.todo-item.selected'); if (selected) selected.scrollIntoView({ behavior: 'smooth', block: 'nearest' }); }
        function formatDate(dateStr) { return new Date(dateStr).toLocaleDateString('en-US', { month: 'short', day: 'numeric', year: 'numeric' }); }
        function escapeHtml(text) { const div = document.createElement('div'); div.textContent = text; return div.innerHTML; }
        function showToast(message, type = 'success') { const toast = document.getElementById('toast'); toast.className = 'toast ' + type + ' show'; document.getElementById('toast-message').textContent = message; setTimeout(() => toast.classList.remove('show'), 3000); }
        setInterval(loadTodos, 10000);
    </script>
</body>
</html>`

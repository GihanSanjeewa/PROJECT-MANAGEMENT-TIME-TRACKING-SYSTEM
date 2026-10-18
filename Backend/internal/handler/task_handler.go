package handler

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"project-management-system/internal/domain"
	"project-management-system/internal/service"
	"project-management-system/pkg/errcode"
	"project-management-system/pkg/middleware"
	"project-management-system/pkg/response"
)

type TaskHandler struct {
	taskService *service.TaskService
}

func NewTaskHandler(taskService *service.TaskService) *TaskHandler {
	return &TaskHandler{taskService: taskService}
}

type CreateTaskRequest struct {
	Title          string  `json:"title"`
	Description    string  `json:"description"`
	RequiredRole   string  `json:"required_role"`
	Priority       string  `json:"priority"`
	EstimatedHours float64 `json:"estimated_hours"`
	StartDate      string  `json:"start_date"`
	DueDate        string  `json:"due_date"`
	AssignedTo     *string `json:"assigned_to"`
	ParentTaskID   *string `json:"parent_task_id"`
}

type UpdateTaskStatusRequest struct {
	Status  string `json:"status"`
	Comment string `json:"comment"`
}

type AddDependencyRequest struct {
	DependsOnTaskID string `json:"depends_on_task_id"`
	DependencyType  string `json:"dependency_type"`
}

// HandleProjectTasks handles GET /api/v1/projects/:id/tasks and POST /api/v1/projects/:id/tasks
func (h *TaskHandler) HandleProjectTasks(w http.ResponseWriter, r *http.Request) {
	reqID := middleware.GetRequestID(r.Context())
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) < 4 || parts[1] != "projects" || parts[3] != "tasks" {
		response.Error(w, errcode.NewNotFound("Resource not found"), reqID)
		return
	}
	projectID := parts[2]

	switch r.Method {
	case http.MethodGet:
		tasks, err := h.taskService.ListTasksByProject(r.Context(), projectID)
		if err != nil {
			response.Error(w, err, reqID)
			return
		}
		response.JSON(w, http.StatusOK, tasks, reqID)

	case http.MethodPost:
		var req CreateTaskRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			response.Error(w, errcode.NewBadRequest("Invalid JSON request body", nil), reqID)
			return
		}

		start, err := time.Parse(time.RFC3339, req.StartDate)
		if err != nil {
			start, err = time.Parse("2006-01-02", req.StartDate)
			if err != nil {
				response.Error(w, errcode.NewValidation("Invalid start_date format, use YYYY-MM-DD or RFC3339", nil), reqID)
				return
			}
		}

		due, err := time.Parse(time.RFC3339, req.DueDate)
		if err != nil {
			due, err = time.Parse("2006-01-02", req.DueDate)
			if err != nil {
				response.Error(w, errcode.NewValidation("Invalid due_date format, use YYYY-MM-DD or RFC3339", nil), reqID)
				return
			}
		}

		task := &domain.Task{
			ProjectID:      projectID,
			ParentTaskID:   req.ParentTaskID,
			Title:          req.Title,
			Description:    req.Description,
			RequiredRole:   domain.RequiredRole(strings.ToUpper(req.RequiredRole)),
			Priority:       domain.Priority(strings.ToUpper(req.Priority)),
			EstimatedHours: req.EstimatedHours,
			StartDate:      start,
			DueDate:        due,
			AssignedTo:     req.AssignedTo,
		}

		created, err := h.taskService.CreateTask(r.Context(), task)
		if err != nil {
			response.Error(w, err, reqID)
			return
		}
		response.JSON(w, http.StatusCreated, created, reqID)

	default:
		response.Error(w, errcode.New(errcode.ErrBadRequest, "Method not allowed", http.StatusMethodNotAllowed, nil), reqID)
	}
}

// HandleTaskByID handles GET, PUT, DELETE on /api/v1/tasks/:id
func (h *TaskHandler) HandleTaskByID(w http.ResponseWriter, r *http.Request) {
	reqID := middleware.GetRequestID(r.Context())
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) < 3 || parts[1] != "tasks" {
		response.Error(w, errcode.NewNotFound("Task endpoint not found"), reqID)
		return
	}
	taskID := parts[2]

	// Handle sub-resources: /api/v1/tasks/:id/status, /api/v1/tasks/:id/audit, /api/v1/tasks/:id/dependencies
	if len(parts) >= 4 {
		sub := parts[3]
		switch sub {
		case "status":
			h.handleTaskStatus(w, r, taskID, reqID)
			return
		case "audit":
			h.handleTaskAudit(w, r, taskID, reqID)
			return
		case "dependencies":
			h.handleTaskDependencies(w, r, taskID, reqID)
			return
		default:
			response.Error(w, errcode.NewNotFound("Unknown task sub-resource"), reqID)
			return
		}
	}

	switch r.Method {
	case http.MethodGet:
		task, err := h.taskService.GetTaskByID(r.Context(), taskID)
		if err != nil {
			response.Error(w, err, reqID)
			return
		}
		response.JSON(w, http.StatusOK, task, reqID)

	case http.MethodPut:
		var req CreateTaskRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			response.Error(w, errcode.NewBadRequest("Invalid JSON request body", nil), reqID)
			return
		}

		update := &domain.Task{
			Title:          req.Title,
			Description:    req.Description,
			RequiredRole:   domain.RequiredRole(strings.ToUpper(req.RequiredRole)),
			Priority:       domain.Priority(strings.ToUpper(req.Priority)),
			EstimatedHours: req.EstimatedHours,
			AssignedTo:     req.AssignedTo,
		}

		if req.StartDate != "" {
			start, err := time.Parse("2006-01-02", req.StartDate)
			if err != nil {
				start, _ = time.Parse(time.RFC3339, req.StartDate)
			}
			update.StartDate = start
		}
		if req.DueDate != "" {
			due, err := time.Parse("2006-01-02", req.DueDate)
			if err != nil {
				due, _ = time.Parse(time.RFC3339, req.DueDate)
			}
			update.DueDate = due
		}

		updated, err := h.taskService.UpdateTask(r.Context(), taskID, update)
		if err != nil {
			response.Error(w, err, reqID)
			return
		}
		response.JSON(w, http.StatusOK, updated, reqID)

	case http.MethodDelete:
		if err := h.taskService.DeleteTask(r.Context(), taskID); err != nil {
			response.Error(w, err, reqID)
			return
		}
		response.JSON(w, http.StatusOK, map[string]string{"message": "Task deleted successfully"}, reqID)

	default:
		response.Error(w, errcode.New(errcode.ErrBadRequest, "Method not allowed", http.StatusMethodNotAllowed, nil), reqID)
	}
}

// handleTaskStatus handles PATCH /api/v1/tasks/:id/status (TSK-021)
func (h *TaskHandler) handleTaskStatus(w http.ResponseWriter, r *http.Request, taskID, reqID string) {
	if r.Method != http.MethodPatch && r.Method != http.MethodPut {
		response.Error(w, errcode.New(errcode.ErrBadRequest, "Method not allowed", http.StatusMethodNotAllowed, nil), reqID)
		return
	}

	var req UpdateTaskStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, errcode.NewBadRequest("Invalid JSON request body", nil), reqID)
		return
	}

	claims := middleware.GetAuthClaims(r.Context())
	userID := "system"
	userRole := "DEVELOPER"
	if claims != nil {
		userID = claims.UserID
		userRole = string(claims.SystemRole)
	}

	updated, audit, err := h.taskService.UpdateTaskStatus(
		r.Context(),
		taskID,
		domain.TaskStatus(strings.ToUpper(req.Status)),
		userID,
		userRole,
		req.Comment,
	)
	if err != nil {
		response.Error(w, err, reqID)
		return
	}

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"task":  updated,
		"audit": audit,
	}, reqID)
}

// handleTaskAudit handles GET /api/v1/tasks/:id/audit (TSK-021)
func (h *TaskHandler) handleTaskAudit(w http.ResponseWriter, r *http.Request, taskID, reqID string) {
	if r.Method != http.MethodGet {
		response.Error(w, errcode.New(errcode.ErrBadRequest, "Method not allowed", http.StatusMethodNotAllowed, nil), reqID)
		return
	}

	logs, err := h.taskService.GetTaskAuditLogs(r.Context(), taskID)
	if err != nil {
		response.Error(w, err, reqID)
		return
	}
	response.JSON(w, http.StatusOK, logs, reqID)
}

// handleTaskDependencies handles POST /api/v1/tasks/:id/dependencies (TSK-019)
func (h *TaskHandler) handleTaskDependencies(w http.ResponseWriter, r *http.Request, taskID, reqID string) {
	if r.Method != http.MethodPost {
		response.Error(w, errcode.New(errcode.ErrBadRequest, "Method not allowed", http.StatusMethodNotAllowed, nil), reqID)
		return
	}

	var req AddDependencyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, errcode.NewBadRequest("Invalid JSON request body", nil), reqID)
		return
	}

	task, err := h.taskService.GetTaskByID(r.Context(), taskID)
	if err != nil {
		response.Error(w, err, reqID)
		return
	}

	dep, err := h.taskService.AddDependency(r.Context(), task.ProjectID, taskID, req.DependsOnTaskID, req.DependencyType)
	if err != nil {
		response.Error(w, err, reqID)
		return
	}

	response.JSON(w, http.StatusCreated, dep, reqID)
}

// HandleProjectWorkload handles GET /api/v1/projects/:id/workload (TSK-020)
func (h *TaskHandler) HandleProjectWorkload(w http.ResponseWriter, r *http.Request) {
	reqID := middleware.GetRequestID(r.Context())
	if r.Method != http.MethodGet {
		response.Error(w, errcode.New(errcode.ErrBadRequest, "Method not allowed", http.StatusMethodNotAllowed, nil), reqID)
		return
	}

	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) < 4 || parts[1] != "projects" || parts[3] != "workload" {
		response.Error(w, errcode.NewNotFound("Workload endpoint not found"), reqID)
		return
	}
	projectID := parts[2]

	workloads, err := h.taskService.GetProjectWorkload(r.Context(), projectID)
	if err != nil {
		response.Error(w, err, reqID)
		return
	}

	response.JSON(w, http.StatusOK, workloads, reqID)
}

// HandleProjectDependencies handles GET /api/v1/projects/:id/dependencies (TSK-019)
func (h *TaskHandler) HandleProjectDependencies(w http.ResponseWriter, r *http.Request) {
	reqID := middleware.GetRequestID(r.Context())
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) < 4 || parts[1] != "projects" || parts[3] != "dependencies" {
		response.Error(w, errcode.NewNotFound("Dependencies endpoint not found"), reqID)
		return
	}
	projectID := parts[2]

	if r.Method == http.MethodGet {
		deps, err := h.taskService.GetProjectDependencies(r.Context(), projectID)
		if err != nil {
			response.Error(w, err, reqID)
			return
		}
		response.JSON(w, http.StatusOK, deps, reqID)
		return
	}

	response.Error(w, errcode.New(errcode.ErrBadRequest, "Method not allowed", http.StatusMethodNotAllowed, nil), reqID)
}

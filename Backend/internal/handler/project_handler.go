package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"project-management-system/internal/domain"
	"project-management-system/internal/service"
	"project-management-system/pkg/errcode"
	"project-management-system/pkg/middleware"
	"project-management-system/pkg/response"
)

type ProjectHandler struct {
	projectService     *service.ProjectService
	taskHandler        *TaskHandler
	feasibilityHandler *FeasibilityHandler
	timeLogHandler     *TimeLogHandler
}

func NewProjectHandler(projectService *service.ProjectService) *ProjectHandler {
	return &ProjectHandler{projectService: projectService}
}

func (h *ProjectHandler) SetTaskHandler(taskHandler *TaskHandler) {
	h.taskHandler = taskHandler
}

func (h *ProjectHandler) SetFeasibilityHandler(feasibilityHandler *FeasibilityHandler) {
	h.feasibilityHandler = feasibilityHandler
}

func (h *ProjectHandler) SetTimeLogHandler(timeLogHandler *TimeLogHandler) {
	h.timeLogHandler = timeLogHandler
}

func (h *ProjectHandler) HandleProjects(w http.ResponseWriter, r *http.Request) {
	reqID := middleware.GetRequestID(r.Context())

	switch r.Method {
	case http.MethodGet:
		projects, err := h.projectService.List(r.Context())
		if err != nil {
			response.Error(w, err, reqID)
			return
		}
		response.JSON(w, http.StatusOK, projects, reqID)

	case http.MethodPost:
		claims := middleware.GetAuthClaims(r.Context())
		if claims != nil && claims.SystemRole != domain.RoleAdmin && claims.SystemRole != domain.RoleProjectManager {
			response.Error(w, errcode.NewForbidden("Only Admins and Project Managers can create projects"), reqID)
			return
		}

		var dto service.CreateProjectDTO
		if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
			response.Error(w, errcode.NewBadRequest("Invalid JSON request body", nil), reqID)
			return
		}

		if claims != nil {
			dto.CreatedBy = claims.UserID
		}

		project, err := h.projectService.Create(r.Context(), dto)
		if err != nil {
			response.Error(w, err, reqID)
			return
		}

		response.JSON(w, http.StatusCreated, project, reqID)

	default:
		response.Error(w, errcode.New(errcode.ErrBadRequest, "Method not allowed", http.StatusMethodNotAllowed, nil), reqID)
	}
}

func (h *ProjectHandler) HandleProjectByID(w http.ResponseWriter, r *http.Request) {
	reqID := middleware.GetRequestID(r.Context())
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/projects/")
	parts := strings.Split(path, "/")
	projectID := parts[0]

	if projectID == "" {
		response.Error(w, errcode.NewBadRequest("Project ID required", nil), reqID)
		return
	}

	// Route /api/v1/projects/:id/members
	if len(parts) >= 2 && parts[1] == "members" {
		h.handleMembersRoute(w, r, projectID, parts)
		return
	}

	// Route /api/v1/projects/:id/capacity
	if len(parts) >= 2 && parts[1] == "capacity" {
		capDTO, err := h.projectService.CalculateTeamCapacity(r.Context(), projectID)
		if err != nil {
			response.Error(w, err, reqID)
			return
		}
		response.JSON(w, http.StatusOK, capDTO, reqID)
		return
	}

	// Route /api/v1/projects/:id/tasks
	if len(parts) >= 2 && parts[1] == "tasks" && h.taskHandler != nil {
		h.taskHandler.HandleProjectTasks(w, r)
		return
	}

	// Route /api/v1/projects/:id/workload
	if len(parts) >= 2 && parts[1] == "workload" && h.taskHandler != nil {
		h.taskHandler.HandleProjectWorkload(w, r)
		return
	}

	// Route /api/v1/projects/:id/dependencies
	if len(parts) >= 2 && parts[1] == "dependencies" && h.taskHandler != nil {
		h.taskHandler.HandleProjectDependencies(w, r)
		return
	}

	// Route /api/v1/projects/:id/feasibility
	if len(parts) >= 2 && parts[1] == "feasibility" && h.feasibilityHandler != nil {
		h.feasibilityHandler.HandleProjectFeasibility(w, r)
		return
	}

	// Route /api/v1/projects/:id/timelogs
	if len(parts) >= 2 && parts[1] == "timelogs" && h.timeLogHandler != nil {
		h.timeLogHandler.HandleProjectTimeLogs(w, r, projectID)
		return
	}

	// Route /api/v1/projects/:id/burndown
	if len(parts) >= 2 && parts[1] == "burndown" && h.timeLogHandler != nil {
		h.timeLogHandler.HandleProjectBurndown(w, r, projectID)
		return
	}

	switch r.Method {
	case http.MethodGet:
		project, err := h.projectService.GetByID(r.Context(), projectID)
		if err != nil {
			response.Error(w, err, reqID)
			return
		}
		response.JSON(w, http.StatusOK, project, reqID)

	case http.MethodPut, http.MethodPatch:
		claims := middleware.GetAuthClaims(r.Context())
		if claims != nil && claims.SystemRole != domain.RoleAdmin && claims.SystemRole != domain.RoleProjectManager {
			response.Error(w, errcode.NewForbidden("Only Admins and Project Managers can modify project timelines"), reqID)
			return
		}

		var dto service.UpdateProjectDTO
		if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
			response.Error(w, errcode.NewBadRequest("Invalid JSON request body", nil), reqID)
			return
		}

		project, err := h.projectService.Update(r.Context(), projectID, dto)
		if err != nil {
			response.Error(w, err, reqID)
			return
		}

		response.JSON(w, http.StatusOK, project, reqID)

	default:
		response.Error(w, errcode.New(errcode.ErrBadRequest, "Method not allowed", http.StatusMethodNotAllowed, nil), reqID)
	}
}

func (h *ProjectHandler) handleMembersRoute(w http.ResponseWriter, r *http.Request, projectID string, parts []string) {
	reqID := middleware.GetRequestID(r.Context())

	switch r.Method {
	case http.MethodGet:
		members, err := h.projectService.GetMembers(r.Context(), projectID)
		if err != nil {
			response.Error(w, err, reqID)
			return
		}
		response.JSON(w, http.StatusOK, members, reqID)

	case http.MethodPost:
		claims := middleware.GetAuthClaims(r.Context())
		if claims != nil && claims.SystemRole != domain.RoleAdmin && claims.SystemRole != domain.RoleProjectManager {
			response.Error(w, errcode.NewForbidden("Only Admins and Project Managers can assign team members"), reqID)
			return
		}

		var dto service.AssignMemberDTO
		if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
			response.Error(w, errcode.NewBadRequest("Invalid JSON request body", nil), reqID)
			return
		}
		dto.ProjectID = projectID

		member, err := h.projectService.AddMember(r.Context(), dto)
		if err != nil {
			response.Error(w, err, reqID)
			return
		}
		response.JSON(w, http.StatusCreated, member, reqID)

	case http.MethodDelete:
		claims := middleware.GetAuthClaims(r.Context())
		if claims != nil && claims.SystemRole != domain.RoleAdmin && claims.SystemRole != domain.RoleProjectManager {
			response.Error(w, errcode.NewForbidden("Only Admins and Project Managers can remove team members"), reqID)
			return
		}

		if len(parts) < 3 || parts[2] == "" {
			response.Error(w, errcode.NewBadRequest("User ID to remove is required in URL", nil), reqID)
			return
		}
		userID := parts[2]

		if err := h.projectService.RemoveMember(r.Context(), projectID, userID); err != nil {
			response.Error(w, err, reqID)
			return
		}
		response.JSON(w, http.StatusOK, map[string]string{"message": "Member removed successfully"}, reqID)

	default:
		response.Error(w, errcode.New(errcode.ErrBadRequest, "Method not allowed", http.StatusMethodNotAllowed, nil), reqID)
	}
}

package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"project-management-system/internal/domain"
	"project-management-system/internal/service"
	"project-management-system/pkg/errcode"
	"project-management-system/pkg/middleware"
	"project-management-system/pkg/response"
)

type TimeLogHandler struct {
	timeLogService *service.TimeLogService
}

func NewTimeLogHandler(timeLogService *service.TimeLogService) *TimeLogHandler {
	return &TimeLogHandler{timeLogService: timeLogService}
}

type StartTimerRequest struct {
	Description string `json:"description"`
}

type CreateManualLogRequest struct {
	WorkDate        string `json:"work_date"`
	StartTime       string `json:"start_time,omitempty"`
	EndTime         string `json:"end_time,omitempty"`
	DurationMinutes int    `json:"duration_minutes,omitempty"`
	Description     string `json:"description"`
}

// HandleTimer handles /api/v1/timer/stop and /api/v1/timer/active
func (h *TimeLogHandler) HandleTimer(w http.ResponseWriter, r *http.Request) {
	reqID := middleware.GetRequestID(r.Context())
	claims := middleware.GetAuthClaims(r.Context())
	if claims == nil {
		response.Error(w, errcode.NewUnauthorized("Authentication required"), reqID)
		return
	}

	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) < 3 || parts[1] != "timer" {
		response.Error(w, errcode.NewNotFound("Resource not found"), reqID)
		return
	}

	action := parts[2]
	switch action {
	case "active":
		if r.Method != http.MethodGet {
			response.Error(w, errcode.New(errcode.ErrBadRequest, "Method not allowed", http.StatusMethodNotAllowed, nil), reqID)
			return
		}
		active, err := h.timeLogService.GetActiveTimer(r.Context(), claims.UserID)
		if err != nil {
			response.Error(w, err, reqID)
			return
		}
		response.JSON(w, http.StatusOK, active, reqID)

	case "stop":
		if r.Method != http.MethodPost {
			response.Error(w, errcode.New(errcode.ErrBadRequest, "Method not allowed", http.StatusMethodNotAllowed, nil), reqID)
			return
		}
		stopped, err := h.timeLogService.StopTimer(r.Context(), claims.UserID)
		if err != nil {
			response.Error(w, err, reqID)
			return
		}
		response.JSON(w, http.StatusOK, stopped, reqID)

	default:
		response.Error(w, errcode.NewNotFound("Unknown timer action"), reqID)
	}
}

// HandleTaskTimer handles POST /api/v1/tasks/:id/timer/start
func (h *TimeLogHandler) HandleTaskTimer(w http.ResponseWriter, r *http.Request, taskID string) {
	reqID := middleware.GetRequestID(r.Context())
	claims := middleware.GetAuthClaims(r.Context())
	if claims == nil {
		response.Error(w, errcode.NewUnauthorized("Authentication required"), reqID)
		return
	}

	if r.Method != http.MethodPost {
		response.Error(w, errcode.New(errcode.ErrBadRequest, "Method not allowed", http.StatusMethodNotAllowed, nil), reqID)
		return
	}

	var req StartTimerRequest
	if r.Body != nil && r.ContentLength > 0 {
		_ = json.NewDecoder(r.Body).Decode(&req)
	}

	log, err := h.timeLogService.StartTimer(r.Context(), taskID, claims.UserID, req.Description)
	if err != nil {
		response.Error(w, err, reqID)
		return
	}

	response.JSON(w, http.StatusCreated, log, reqID)
}

// HandleTaskTimeLogs handles GET and POST on /api/v1/tasks/:id/timelogs
func (h *TimeLogHandler) HandleTaskTimeLogs(w http.ResponseWriter, r *http.Request, taskID string) {
	reqID := middleware.GetRequestID(r.Context())
	claims := middleware.GetAuthClaims(r.Context())
	if claims == nil {
		response.Error(w, errcode.NewUnauthorized("Authentication required"), reqID)
		return
	}

	switch r.Method {
	case http.MethodGet:
		logs, err := h.timeLogService.ListLogsByTask(r.Context(), taskID)
		if err != nil {
			response.Error(w, err, reqID)
			return
		}
		response.JSON(w, http.StatusOK, logs, reqID)

	case http.MethodPost:
		var req CreateManualLogRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			response.Error(w, errcode.NewBadRequest("Invalid JSON request body", nil), reqID)
			return
		}

		timeLog := &domain.TimeLog{
			TaskID:          taskID,
			UserID:          claims.UserID,
			DurationMinutes: req.DurationMinutes,
			Description:     req.Description,
			WorkDate:        req.WorkDate,
		}

		if req.StartTime != "" {
			st, err := time.Parse(time.RFC3339, req.StartTime)
			if err != nil {
				st, _ = time.Parse("2006-01-02 15:04:05", req.StartTime)
			}
			timeLog.StartTime = st
		}
		if req.EndTime != "" {
			et, err := time.Parse(time.RFC3339, req.EndTime)
			if err != nil {
				et, _ = time.Parse("2006-01-02 15:04:05", req.EndTime)
			}
			timeLog.EndTime = &et
		}

		created, err := h.timeLogService.CreateManualLog(r.Context(), timeLog)
		if err != nil {
			response.Error(w, err, reqID)
			return
		}
		response.JSON(w, http.StatusCreated, created, reqID)

	default:
		response.Error(w, errcode.New(errcode.ErrBadRequest, "Method not allowed", http.StatusMethodNotAllowed, nil), reqID)
	}
}

// HandleTimeLogByID handles PUT and DELETE on /api/v1/timelogs/:id
func (h *TimeLogHandler) HandleTimeLogByID(w http.ResponseWriter, r *http.Request) {
	reqID := middleware.GetRequestID(r.Context())
	claims := middleware.GetAuthClaims(r.Context())
	if claims == nil {
		response.Error(w, errcode.NewUnauthorized("Authentication required"), reqID)
		return
	}

	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) < 3 || parts[1] != "timelogs" {
		response.Error(w, errcode.NewNotFound("Resource not found"), reqID)
		return
	}
	logID := parts[2]
	isAdminOrPM := claims.SystemRole == domain.RoleAdmin || claims.SystemRole == domain.RoleProjectManager

	switch r.Method {
	case http.MethodPut:
		var req CreateManualLogRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			response.Error(w, errcode.NewBadRequest("Invalid JSON request body", nil), reqID)
			return
		}

		update := &domain.TimeLog{
			DurationMinutes: req.DurationMinutes,
			Description:     req.Description,
			WorkDate:        req.WorkDate,
		}
		if req.StartTime != "" {
			st, err := time.Parse(time.RFC3339, req.StartTime)
			if err != nil {
				st, _ = time.Parse("2006-01-02 15:04:05", req.StartTime)
			}
			update.StartTime = st
		}
		if req.EndTime != "" {
			et, err := time.Parse(time.RFC3339, req.EndTime)
			if err != nil {
				et, _ = time.Parse("2006-01-02 15:04:05", req.EndTime)
			}
			update.EndTime = &et
		}

		updated, err := h.timeLogService.UpdateManualLog(r.Context(), logID, update, claims.UserID, isAdminOrPM)
		if err != nil {
			response.Error(w, err, reqID)
			return
		}
		response.JSON(w, http.StatusOK, updated, reqID)

	case http.MethodDelete:
		if err := h.timeLogService.DeleteLog(r.Context(), logID, claims.UserID, isAdminOrPM); err != nil {
			response.Error(w, err, reqID)
			return
		}
		response.JSON(w, http.StatusOK, map[string]string{"message": "Time log deleted successfully"}, reqID)

	default:
		response.Error(w, errcode.New(errcode.ErrBadRequest, "Method not allowed", http.StatusMethodNotAllowed, nil), reqID)
	}
}

// HandleProjectTimeLogs handles GET /api/v1/projects/:id/timelogs
func (h *TimeLogHandler) HandleProjectTimeLogs(w http.ResponseWriter, r *http.Request, projectID string) {
	reqID := middleware.GetRequestID(r.Context())
	if r.Method != http.MethodGet {
		response.Error(w, errcode.New(errcode.ErrBadRequest, "Method not allowed", http.StatusMethodNotAllowed, nil), reqID)
		return
	}

	logs, err := h.timeLogService.ListLogsByProject(r.Context(), projectID)
	if err != nil {
		response.Error(w, err, reqID)
		return
	}
	response.JSON(w, http.StatusOK, logs, reqID)
}

// HandleProjectBurndown handles GET /api/v1/projects/:id/burndown
func (h *TimeLogHandler) HandleProjectBurndown(w http.ResponseWriter, r *http.Request, projectID string) {
	reqID := middleware.GetRequestID(r.Context())
	if r.Method != http.MethodGet {
		response.Error(w, errcode.New(errcode.ErrBadRequest, "Method not allowed", http.StatusMethodNotAllowed, nil), reqID)
		return
	}

	tzOffset := parseTzOffset(r.URL.Query().Get("tz"))
	burndown, err := h.timeLogService.GetProjectBurndown(r.Context(), projectID, tzOffset)
	if err != nil {
		response.Error(w, err, reqID)
		return
	}
	response.JSON(w, http.StatusOK, burndown, reqID)
}

// HandleWeeklyTimesheet handles GET /api/v1/timesheets/weekly
func (h *TimeLogHandler) HandleWeeklyTimesheet(w http.ResponseWriter, r *http.Request) {
	reqID := middleware.GetRequestID(r.Context())
	claims := middleware.GetAuthClaims(r.Context())
	if claims == nil {
		response.Error(w, errcode.NewUnauthorized("Authentication required"), reqID)
		return
	}

	if r.Method != http.MethodGet {
		response.Error(w, errcode.New(errcode.ErrBadRequest, "Method not allowed", http.StatusMethodNotAllowed, nil), reqID)
		return
	}

	targetUserID := r.URL.Query().Get("user_id")
	if targetUserID == "" {
		targetUserID = claims.UserID
	}

	weekStart := r.URL.Query().Get("week_start")
	tzOffset := parseTzOffset(r.URL.Query().Get("tz"))

	timesheet, err := h.timeLogService.GetWeeklyTimesheet(r.Context(), targetUserID, weekStart, tzOffset)
	if err != nil {
		response.Error(w, err, reqID)
		return
	}
	response.JSON(w, http.StatusOK, timesheet, reqID)
}

func parseTzOffset(tz string) int {
	if tz == "" {
		return 0
	}
	if offset, err := strconv.Atoi(tz); err == nil {
		return offset
	}
	// Named timezone
	if loc, err := time.LoadLocation(tz); err == nil {
		_, offsetSec := time.Now().In(loc).Zone()
		return offsetSec / 60
	}
	return 0
}

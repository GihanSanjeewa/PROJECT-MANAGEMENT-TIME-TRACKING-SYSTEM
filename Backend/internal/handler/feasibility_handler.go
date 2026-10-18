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

type FeasibilityHandler struct {
	feasibilityService *service.FeasibilityService
}

func NewFeasibilityHandler(feasibilityService *service.FeasibilityService) *FeasibilityHandler {
	return &FeasibilityHandler{feasibilityService: feasibilityService}
}

// HandleProjectFeasibility handles GET /api/v1/projects/:id/feasibility and POST /api/v1/projects/:id/feasibility/simulate
func (h *FeasibilityHandler) HandleProjectFeasibility(w http.ResponseWriter, r *http.Request) {
	reqID := middleware.GetRequestID(r.Context())
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) < 4 || parts[1] != "projects" || parts[3] != "feasibility" {
		response.Error(w, errcode.NewNotFound("Feasibility endpoint not found"), reqID)
		return
	}
	projectID := parts[2]

	// Check if this is the simulation endpoint: /api/v1/projects/:id/feasibility/simulate
	if len(parts) >= 5 && parts[4] == "simulate" {
		if r.Method != http.MethodPost {
			response.Error(w, errcode.New(errcode.ErrBadRequest, "Method not allowed", http.StatusMethodNotAllowed, nil), reqID)
			return
		}

		var req domain.SimulationRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			response.Error(w, errcode.NewBadRequest("Invalid JSON simulation payload", nil), reqID)
			return
		}

		result, err := h.feasibilityService.SimulateWhatIf(r.Context(), projectID, req)
		if err != nil {
			response.Error(w, err, reqID)
			return
		}

		response.JSON(w, http.StatusOK, result, reqID)
		return
	}

	// Default: GET /api/v1/projects/:id/feasibility
	if r.Method == http.MethodGet {
		report, err := h.feasibilityService.EvaluateProject(r.Context(), projectID)
		if err != nil {
			response.Error(w, err, reqID)
			return
		}
		response.JSON(w, http.StatusOK, report, reqID)
		return
	}

	response.Error(w, errcode.New(errcode.ErrBadRequest, "Method not allowed", http.StatusMethodNotAllowed, nil), reqID)
}

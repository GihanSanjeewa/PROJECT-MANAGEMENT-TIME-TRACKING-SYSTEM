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

type UserHandler struct {
	userService *service.UserService
}

func NewUserHandler(userService *service.UserService) *UserHandler {
	return &UserHandler{userService: userService}
}

func (h *UserHandler) HandleUsers(w http.ResponseWriter, r *http.Request) {
	reqID := middleware.GetRequestID(r.Context())

	switch r.Method {
	case http.MethodGet:
		// Filter by ?role=
		roleQuery := r.URL.Query().Get("role")
		var filterRole domain.SystemRole
		if roleQuery != "" {
			filterRole = domain.SystemRole(strings.ToUpper(roleQuery))
		}

		users, err := h.userService.List(r.Context(), filterRole)
		if err != nil {
			response.Error(w, err, reqID)
			return
		}
		response.JSON(w, http.StatusOK, users, reqID)

	case http.MethodPost:
		// Create User (Restricted to Admin / Project Manager)
		claims := middleware.GetAuthClaims(r.Context())
		if claims == nil || (claims.SystemRole != domain.RoleAdmin && claims.SystemRole != domain.RoleProjectManager) {
			response.Error(w, errcode.NewForbidden("Only Admins and Project Managers can register new users"), reqID)
			return
		}

		var dto service.CreateUserDTO
		if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
			response.Error(w, errcode.NewBadRequest("Invalid JSON request body", nil), reqID)
			return
		}

		user, err := h.userService.Create(r.Context(), dto)
		if err != nil {
			response.Error(w, err, reqID)
			return
		}
		response.JSON(w, http.StatusCreated, user, reqID)

	default:
		response.Error(w, errcode.New(errcode.ErrBadRequest, "Method not allowed", http.StatusMethodNotAllowed, nil), reqID)
	}
}

func (h *UserHandler) HandleUserByID(w http.ResponseWriter, r *http.Request) {
	reqID := middleware.GetRequestID(r.Context())
	id := strings.TrimPrefix(r.URL.Path, "/api/v1/users/")
	if id == "" {
		response.Error(w, errcode.NewBadRequest("User ID required", nil), reqID)
		return
	}

	switch r.Method {
	case http.MethodGet:
		user, err := h.userService.GetByID(r.Context(), id)
		if err != nil {
			response.Error(w, err, reqID)
			return
		}
		response.JSON(w, http.StatusOK, user, reqID)

	case http.MethodPatch, http.MethodPut:
		// Only self or Admin/PM can update user details
		claims := middleware.GetAuthClaims(r.Context())
		if claims == nil || (claims.UserID != id && claims.SystemRole != domain.RoleAdmin && claims.SystemRole != domain.RoleProjectManager) {
			response.Error(w, errcode.NewForbidden("You do not have permission to update this user profile"), reqID)
			return
		}

		var dto service.UpdateUserDTO
		if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
			response.Error(w, errcode.NewBadRequest("Invalid JSON request body", nil), reqID)
			return
		}

		user, err := h.userService.Update(r.Context(), id, dto)
		if err != nil {
			response.Error(w, err, reqID)
			return
		}
		response.JSON(w, http.StatusOK, user, reqID)

	default:
		response.Error(w, errcode.New(errcode.ErrBadRequest, "Method not allowed", http.StatusMethodNotAllowed, nil), reqID)
	}
}

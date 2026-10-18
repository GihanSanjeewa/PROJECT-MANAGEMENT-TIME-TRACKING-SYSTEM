package handler

import (
	"encoding/json"
	"net/http"

	"project-management-system/internal/service"
	"project-management-system/pkg/errcode"
	"project-management-system/pkg/middleware"
	"project-management-system/pkg/response"
)

type AuthHandler struct {
	authService *service.AuthService
	userService *service.UserService
}

func NewAuthHandler(authService *service.AuthService, userService *service.UserService) *AuthHandler {
	return &AuthHandler{
		authService: authService,
		userService: userService,
	}
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	reqID := middleware.GetRequestID(r.Context())
	if r.Method != http.MethodPost {
		response.Error(w, errcode.New(errcode.ErrBadRequest, "Method not allowed", http.StatusMethodNotAllowed, nil), reqID)
		return
	}

	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, errcode.NewBadRequest("Invalid JSON request body", nil), reqID)
		return
	}

	clientIP := r.RemoteAddr
	user, tokens, err := h.authService.Login(r.Context(), req.Email, req.Password, clientIP)
	if err != nil {
		response.Error(w, err, reqID)
		return
	}

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"user":          user,
		"access_token":  tokens.AccessToken,
		"refresh_token": tokens.RefreshToken,
		"expires_in":    tokens.ExpiresIn,
	}, reqID)
}

func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	reqID := middleware.GetRequestID(r.Context())
	if r.Method != http.MethodPost {
		response.Error(w, errcode.New(errcode.ErrBadRequest, "Method not allowed", http.StatusMethodNotAllowed, nil), reqID)
		return
	}

	var req RefreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.RefreshToken == "" {
		response.Error(w, errcode.NewBadRequest("refresh_token is required", nil), reqID)
		return
	}

	tokens, err := h.authService.RefreshToken(r.Context(), req.RefreshToken)
	if err != nil {
		response.Error(w, err, reqID)
		return
	}

	response.JSON(w, http.StatusOK, tokens, reqID)
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	reqID := middleware.GetRequestID(r.Context())
	if r.Method != http.MethodPost {
		response.Error(w, errcode.New(errcode.ErrBadRequest, "Method not allowed", http.StatusMethodNotAllowed, nil), reqID)
		return
	}

	var req RefreshRequest
	_ = json.NewDecoder(r.Body).Decode(&req)
	if req.RefreshToken != "" {
		_ = h.authService.Logout(r.Context(), req.RefreshToken)
	}

	response.JSON(w, http.StatusOK, map[string]string{"message": "Logged out successfully"}, reqID)
}

func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	reqID := middleware.GetRequestID(r.Context())
	claims := middleware.GetAuthClaims(r.Context())
	if claims == nil {
		response.Error(w, errcode.NewUnauthorized("Authentication required"), reqID)
		return
	}

	user, err := h.userService.GetByID(r.Context(), claims.UserID)
	if err != nil {
		response.Error(w, err, reqID)
		return
	}

	response.JSON(w, http.StatusOK, user, reqID)
}

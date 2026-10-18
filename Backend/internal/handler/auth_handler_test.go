package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"project-management-system/internal/domain"
	"project-management-system/internal/handler"
	"project-management-system/internal/repository"
	"project-management-system/internal/service"
	"project-management-system/pkg/middleware"
	"project-management-system/pkg/response"
)

func setupTestServer() (http.Handler, *service.UserService, *service.AuthService) {
	userRepo := repository.NewInMemoryUserRepository()
	tokenStore := repository.NewInMemoryTokenStore()
	jwtSecret := "super-secure-test-jwt-secret-with-at-least-32-chars!"

	authService := service.NewAuthService(userRepo, tokenStore, jwtSecret)
	userService := service.NewUserService(userRepo, authService)

	authHandler := handler.NewAuthHandler(authService, userService)
	userHandler := handler.NewUserHandler(userService)

	mux := http.NewServeMux()

	mux.HandleFunc("/api/v1/auth/login", authHandler.Login)
	mux.HandleFunc("/api/v1/auth/refresh", authHandler.Refresh)
	mux.HandleFunc("/api/v1/auth/logout", authHandler.Logout)

	authMiddleware := middleware.Authenticate(authService)
	mux.Handle("/api/v1/auth/me", authMiddleware(http.HandlerFunc(authHandler.Me)))
	mux.Handle("/api/v1/users", authMiddleware(http.HandlerFunc(userHandler.HandleUsers)))

	pipeline := middleware.RequestID(middleware.Recovery(mux))
	return pipeline, userService, authService
}

func TestAuthEndToEndLifecycle(t *testing.T) {
	server, userService, _ := setupTestServer()
	ctx := context.Background()

	// 1. Create a user
	testUser, err := userService.Create(ctx, service.CreateUserDTO{
		Email:       "maya.pm@pulseflow.io",
		Password:    "SecurePass123!",
		FullName:    "Maya Patel",
		PrimaryRole: domain.RoleProjectManager,
	})
	if err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	// 2. Perform Login
	loginBody, _ := json.Marshal(map[string]string{
		"email":    testUser.Email,
		"password": "SecurePass123!",
	})
	reqLogin := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(loginBody))
	reqLogin.Header.Set("Content-Type", "application/json")
	rrLogin := httptest.NewRecorder()

	server.ServeHTTP(rrLogin, reqLogin)

	if rrLogin.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for login, got %d. Body: %s", rrLogin.Code, rrLogin.Body.String())
	}

	var loginEnv struct {
		Success bool `json:"success"`
		Data    struct {
			AccessToken  string `json:"access_token"`
			RefreshToken string `json:"refresh_token"`
			ExpiresIn    int    `json:"expires_in"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rrLogin.Body.Bytes(), &loginEnv); err != nil {
		t.Fatalf("failed to parse login response: %v", err)
	}

	accessToken := loginEnv.Data.AccessToken
	refreshToken := loginEnv.Data.RefreshToken

	if accessToken == "" || refreshToken == "" {
		t.Fatalf("expected non-empty access and refresh tokens")
	}

	// 3. Request /me with Bearer token
	reqMe := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	reqMe.Header.Set("Authorization", "Bearer "+accessToken)
	rrMe := httptest.NewRecorder()

	server.ServeHTTP(rrMe, reqMe)

	if rrMe.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for /me, got %d. Body: %s", rrMe.Code, rrMe.Body.String())
	}

	var meEnv struct {
		Success bool        `json:"success"`
		Data    domain.User `json:"data"`
	}
	_ = json.Unmarshal(rrMe.Body.Bytes(), &meEnv)
	if meEnv.Data.Email != testUser.Email {
		t.Errorf("expected email '%s', got '%s'", testUser.Email, meEnv.Data.Email)
	}

	// 4. Request /refresh
	refreshBody, _ := json.Marshal(map[string]string{
		"refresh_token": refreshToken,
	})
	reqRefresh := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", bytes.NewReader(refreshBody))
	reqRefresh.Header.Set("Content-Type", "application/json")
	rrRefresh := httptest.NewRecorder()

	server.ServeHTTP(rrRefresh, reqRefresh)

	if rrRefresh.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for /refresh, got %d. Body: %s", rrRefresh.Code, rrRefresh.Body.String())
	}

	var refreshEnv struct {
		Success bool              `json:"success"`
		Data    service.TokenPair `json:"data"`
	}
	_ = json.Unmarshal(rrRefresh.Body.Bytes(), &refreshEnv)

	if refreshEnv.Data.AccessToken == "" {
		t.Errorf("expected new access token from refresh")
	}

	// 5. Logout
	logoutBody, _ := json.Marshal(map[string]string{
		"refresh_token": refreshEnv.Data.RefreshToken,
	})
	reqLogout := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", bytes.NewReader(logoutBody))
	reqLogout.Header.Set("Content-Type", "application/json")
	rrLogout := httptest.NewRecorder()

	server.ServeHTTP(rrLogout, reqLogout)

	if rrLogout.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for /logout, got %d", rrLogout.Code)
	}
}

func TestLoginWrongCredentials(t *testing.T) {
	server, _, _ := setupTestServer()

	loginBody, _ := json.Marshal(map[string]string{
		"email":    "nonexistent@pulseflow.io",
		"password": "WrongPassword!",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(loginBody))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	server.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized, got %d", rr.Code)
	}

	var env response.Envelope
	_ = json.Unmarshal(rr.Body.Bytes(), &env)
	if env.Success {
		t.Errorf("expected success to be false")
	}
	if env.Error.Code != "UNAUTHORIZED" {
		t.Errorf("expected error code 'UNAUTHORIZED', got '%s'", env.Error.Code)
	}
}

package middleware_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"project-management-system/internal/domain"
	"project-management-system/internal/service"
	"project-management-system/pkg/middleware"
)

type mockValidator struct {
	claims *service.CustomClaims
	err    error
}

func (m *mockValidator) ValidateAccessToken(tokenStr string) (*service.CustomClaims, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.claims, nil
}

func TestAuthenticateMiddleware_MissingHeader(t *testing.T) {
	validator := &mockValidator{}
	authMiddleware := middleware.Authenticate(validator)

	nextCalled := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
	})

	handler := middleware.RequestID(authMiddleware(next))
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if nextCalled {
		t.Errorf("expected protected handler not to be called without auth header")
	}
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 Unauthorized, got %d", rr.Code)
	}
}

func TestRequireSystemRole_Authorized(t *testing.T) {
	roleMiddleware := middleware.RequireSystemRole(domain.RoleAdmin, domain.RoleProjectManager)

	nextCalled := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusOK)
	})

	handler := middleware.RequestID(roleMiddleware(next))
	req := httptest.NewRequest(http.MethodPost, "/admin-only", nil)

	// Inject admin claims into context
	claims := &service.CustomClaims{
		UserID:     "admin-1",
		SystemRole: domain.RoleAdmin,
	}
	ctx := context.WithValue(req.Context(), middleware.UserClaimsKey, claims)
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if !nextCalled {
		t.Errorf("expected admin request to be authorized")
	}
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 OK, got %d", rr.Code)
	}
}

func TestRequireSystemRole_Forbidden(t *testing.T) {
	roleMiddleware := middleware.RequireSystemRole(domain.RoleAdmin)

	nextCalled := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
	})

	handler := middleware.RequestID(roleMiddleware(next))
	req := httptest.NewRequest(http.MethodPost, "/admin-only", nil)

	// Inject developer claims into context
	claims := &service.CustomClaims{
		UserID:     "dev-1",
		SystemRole: domain.RoleDeveloper,
	}
	ctx := context.WithValue(req.Context(), middleware.UserClaimsKey, claims)
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if nextCalled {
		t.Errorf("expected developer request to be forbidden from admin endpoint")
	}
	if rr.Code != http.StatusForbidden {
		t.Errorf("expected 403 Forbidden, got %d", rr.Code)
	}
}

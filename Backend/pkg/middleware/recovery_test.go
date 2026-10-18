package middleware_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"project-management-system/pkg/middleware"
	"project-management-system/pkg/response"
)

func TestRecoveryMiddleware(t *testing.T) {
	// A handler that triggers a deliberate nil pointer / panic
	panicHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("deliberate test panic: nil pointer dereference")
	})

	// Wrap with RequestID + Recovery
	handler := middleware.RequestID(middleware.Recovery(panicHandler))

	req := httptest.NewRequest(http.MethodGet, "/test-panic", nil)
	rr := httptest.NewRecorder()

	// Should not crash the test process!
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected HTTP 500 status code, got %d", rr.Code)
	}

	var env response.Envelope
	if err := json.Unmarshal(rr.Body.Bytes(), &env); err != nil {
		t.Fatalf("failed to decode response JSON: %v", err)
	}

	if env.Success {
		t.Errorf("expected success to be false, got true")
	}

	if env.Error == nil {
		t.Fatalf("expected error object to be present in envelope")
	}

	if env.Error.Code != "INTERNAL_ERROR" {
		t.Errorf("expected error code 'INTERNAL_ERROR', got '%s'", env.Error.Code)
	}

	if env.Meta.RequestID == "" {
		t.Errorf("expected meta.request_id to be populated")
	}
}

package middleware

import (
	"context"
	"net/http"
	"strings"

	"project-management-system/internal/domain"
	"project-management-system/internal/service"
	"project-management-system/pkg/errcode"
	"project-management-system/pkg/response"
)

type userContextKey string

const UserClaimsKey userContextKey = "user_claims"

type TokenValidator interface {
	ValidateAccessToken(tokenStr string) (*service.CustomClaims, error)
}

func Authenticate(validator TokenValidator) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			reqID := GetRequestID(r.Context())
			authHeader := r.Header.Get("Authorization")

			if authHeader == "" {
				response.Error(w, errcode.NewUnauthorized("Authorization header is required"), reqID)
				return
			}

			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
				response.Error(w, errcode.NewUnauthorized("Authorization header format must be Bearer {token}"), reqID)
				return
			}

			tokenStr := strings.TrimSpace(parts[1])
			claims, err := validator.ValidateAccessToken(tokenStr)
			if err != nil {
				response.Error(w, err, reqID)
				return
			}

			ctx := context.WithValue(r.Context(), UserClaimsKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func RequireSystemRole(roles ...domain.SystemRole) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			reqID := GetRequestID(r.Context())
			claims := GetAuthClaims(r.Context())

			if claims == nil {
				response.Error(w, errcode.NewUnauthorized("Authentication required"), reqID)
				return
			}

			allowed := false
			for _, r := range roles {
				if claims.SystemRole == r {
					allowed = true
					break
				}
			}

			if !allowed {
				response.Error(w, errcode.NewForbidden("Insufficient privileges to access this resource"), reqID)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func GetAuthClaims(ctx context.Context) *service.CustomClaims {
	if claims, ok := ctx.Value(UserClaimsKey).(*service.CustomClaims); ok {
		return claims
	}
	return nil
}

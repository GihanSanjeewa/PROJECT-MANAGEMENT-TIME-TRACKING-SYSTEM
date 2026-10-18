package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"project-management-system/internal/domain"
	"project-management-system/internal/repository"
	"project-management-system/internal/service"
)

func setupAuthTest() (*service.AuthService, *service.UserService, *repository.InMemoryUserRepository) {
	userRepo := repository.NewInMemoryUserRepository()
	tokenStore := repository.NewInMemoryTokenStore()
	jwtSecret := "super-secure-test-jwt-secret-with-at-least-32-chars!"

	authService := service.NewAuthService(userRepo, tokenStore, jwtSecret)
	userService := service.NewUserService(userRepo, authService)

	return authService, userService, userRepo
}

func TestPasswordHashingAndVerification(t *testing.T) {
	authService, _, _ := setupAuthTest()

	password := "MySecretPass123!"
	hash, err := authService.HashPassword(password)
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}

	if !authService.CheckPassword(hash, password) {
		t.Errorf("expected password to match hash")
	}

	if authService.CheckPassword(hash, "WrongPassword!") {
		t.Errorf("expected wrong password to fail check")
	}
}

func TestTokenPairGenerationAndValidation(t *testing.T) {
	authService, _, _ := setupAuthTest()
	ctx := context.Background()

	user := &domain.User{
		ID:          "usr-test-1",
		Email:       "tester@pulseflow.io",
		PrimaryRole: domain.RoleDeveloper,
	}

	tokens, err := authService.GenerateTokenPair(ctx, user)
	if err != nil {
		t.Fatalf("failed to generate token pair: %v", err)
	}

	if tokens.AccessToken == "" || tokens.RefreshToken == "" {
		t.Fatalf("expected non-empty tokens in pair")
	}

	// Validate Access Token
	claims, err := authService.ValidateAccessToken(tokens.AccessToken)
	if err != nil {
		t.Fatalf("failed to validate access token: %v", err)
	}

	if claims.UserID != user.ID || claims.Email != user.Email || claims.SystemRole != user.PrimaryRole {
		t.Errorf("claims mismatch: %+v", claims)
	}
}

func TestTamperedTokenRejection(t *testing.T) {
	authService, _, _ := setupAuthTest()
	ctx := context.Background()

	user := &domain.User{
		ID:          "usr-test-2",
		Email:       "tamper@pulseflow.io",
		PrimaryRole: domain.RoleQAEngineer,
	}

	tokens, _ := authService.GenerateTokenPair(ctx, user)
	tamperedToken := tokens.AccessToken + "xyz"

	_, err := authService.ValidateAccessToken(tamperedToken)
	if err == nil {
		t.Fatalf("expected error when validating tampered token, got nil")
	}
}

func TestNoneAlgorithmRejection(t *testing.T) {
	authService, _, _ := setupAuthTest()

	// Forge token with "none" algorithm
	claims := jwt.MapClaims{
		"user_id":     "usr-hacker",
		"system_role": "ADMIN",
		"exp":         time.Now().Add(time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodNone, claims)
	noneTokenStr, _ := token.SignedString(jwt.UnsafeAllowNoneSignatureType)

	_, err := authService.ValidateAccessToken(noneTokenStr)
	if err == nil {
		t.Fatalf("CRITICAL SECURITY FLAW: 'none' algorithm token was accepted!")
	}
}

func TestRefreshTokenRotationAndRevocation(t *testing.T) {
	authService, userService, _ := setupAuthTest()
	ctx := context.Background()

	user, err := userService.Create(ctx, service.CreateUserDTO{
		Email:       "rotator@pulseflow.io",
		Password:    "SecurePass123!",
		FullName:    "Token Rotator",
		PrimaryRole: domain.RoleDeveloper,
	})
	if err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	tokens, err := authService.GenerateTokenPair(ctx, user)
	if err != nil {
		t.Fatalf("failed to generate token pair: %v", err)
	}

	// 1. Refreshing with valid refresh token should succeed
	newTokens, err := authService.RefreshToken(ctx, tokens.RefreshToken)
	if err != nil {
		t.Fatalf("failed to refresh token: %v", err)
	}

	if newTokens.AccessToken == "" || newTokens.RefreshToken == "" {
		t.Fatalf("expected new token pair")
	}

	// 2. Reusing old refresh token MUST fail (Single-use rotation rule)
	_, err = authService.RefreshToken(ctx, tokens.RefreshToken)
	if err == nil {
		t.Fatalf("CRITICAL: Reusing an old rotated refresh token should have failed!")
	}
}

func TestUserEnumerationProtection(t *testing.T) {
	authService, userService, _ := setupAuthTest()
	ctx := context.Background()

	_, err := userService.Create(ctx, service.CreateUserDTO{
		Email:       "existing@pulseflow.io",
		Password:    "CorrectPassword123!",
		FullName:    "Existing User",
		PrimaryRole: domain.RoleDeveloper,
	})
	if err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	// 1. Attempt login with non-existent user
	_, _, errNonExistent := authService.Login(ctx, "nonexistent@pulseflow.io", "SomePassword!", "127.0.0.1")

	// 2. Attempt login with real user but wrong password
	_, _, errWrongPass := authService.Login(ctx, "existing@pulseflow.io", "WrongPassword!", "127.0.0.1")

	if errNonExistent == nil || errWrongPass == nil {
		t.Fatalf("expected errors for invalid logins")
	}

	// Both error messages must be strictly identical
	if errNonExistent.Error() != errWrongPass.Error() {
		t.Errorf("User enumeration vulnerability: '%s' vs '%s'", errNonExistent.Error(), errWrongPass.Error())
	}
}

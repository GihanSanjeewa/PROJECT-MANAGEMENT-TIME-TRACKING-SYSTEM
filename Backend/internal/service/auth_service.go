package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"project-management-system/internal/domain"
	"project-management-system/internal/repository"
	"project-management-system/pkg/errcode"
)

const (
	AccessTokenDuration  = 15 * time.Minute
	RefreshTokenDuration = 7 * 24 * time.Hour
	BcryptCost           = 12
	MaxLoginAttempts     = 5
	LockoutDuration      = 1 * time.Minute
)

type CustomClaims struct {
	UserID     string            `json:"user_id"`
	Email      string            `json:"email"`
	SystemRole domain.SystemRole `json:"system_role"`
	jwt.RegisteredClaims
}

type RefreshClaims struct {
	TokenID string `json:"token_id"`
	UserID  string `json:"user_id"`
	jwt.RegisteredClaims
}

type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"` // seconds (900)
}

type rateLimitTracker struct {
	mu       sync.Mutex
	attempts map[string]int
	lockouts map[string]time.Time
}

type AuthService struct {
	userRepo   repository.UserRepository
	tokenStore repository.TokenStore
	jwtSecret  []byte
	limiter    *rateLimitTracker
}

func NewAuthService(userRepo repository.UserRepository, tokenStore repository.TokenStore, jwtSecret string) *AuthService {
	return &AuthService{
		userRepo:   userRepo,
		tokenStore: tokenStore,
		jwtSecret:  []byte(jwtSecret),
		limiter: &rateLimitTracker{
			attempts: make(map[string]int),
			lockouts: make(map[string]time.Time),
		},
	}
}

// HashPassword hashes plain password with bcrypt cost 12
func (s *AuthService) HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), BcryptCost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}
	return string(bytes), nil
}

// CheckPassword checks if password matches hash
func (s *AuthService) CheckPassword(hash, password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// Login authenticates credentials, enforces rate limiting, and issues token pair
func (s *AuthService) Login(ctx context.Context, email, password, clientIP string) (*domain.User, *TokenPair, error) {
	// Rate limiting check
	s.limiter.mu.Lock()
	if lockUntil, locked := s.limiter.lockouts[clientIP]; locked {
		if time.Now().Before(lockUntil) {
			s.limiter.mu.Unlock()
			return nil, nil, errcode.New(errcode.ErrForbidden, "Too many failed login attempts. Please try again later.", 429, nil)
		}
		delete(s.limiter.lockouts, clientIP)
		delete(s.limiter.attempts, clientIP)
	}
	s.limiter.mu.Unlock()

	genericError := errcode.NewUnauthorized("Invalid email or password")

	user, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		s.recordFailedAttempt(clientIP)
		// Return identical generic error to prevent user enumeration
		return nil, nil, genericError
	}

	if !user.IsActive {
		s.recordFailedAttempt(clientIP)
		return nil, nil, genericError
	}

	if !s.CheckPassword(user.PasswordHash, password) {
		s.recordFailedAttempt(clientIP)
		return nil, nil, genericError
	}

	// Login succeeded - clear failed attempts
	s.limiter.mu.Lock()
	delete(s.limiter.attempts, clientIP)
	s.limiter.mu.Unlock()

	tokens, err := s.GenerateTokenPair(ctx, user)
	if err != nil {
		return nil, nil, err
	}

	return user, tokens, nil
}

func (s *AuthService) recordFailedAttempt(clientIP string) {
	s.limiter.mu.Lock()
	defer s.limiter.mu.Unlock()

	s.limiter.attempts[clientIP]++
	if s.limiter.attempts[clientIP] >= MaxLoginAttempts {
		s.limiter.lockouts[clientIP] = time.Now().Add(LockoutDuration)
	}
}

// GenerateTokenPair generates an access token and a refresh token
func (s *AuthService) GenerateTokenPair(ctx context.Context, user *domain.User) (*TokenPair, error) {
	now := time.Now().UTC()
	accessExpiry := now.Add(AccessTokenDuration)
	refreshExpiry := now.Add(RefreshTokenDuration)

	// Access Token
	accessClaims := CustomClaims{
		UserID:     user.ID,
		Email:      user.Email,
		SystemRole: user.PrimaryRole,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   user.ID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(accessExpiry),
			NotBefore: jwt.NewNumericDate(now.Add(-30 * time.Second)), // 30s clock skew tolerance
			Issuer:    "pulseflow-auth",
		},
	}
	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	accessStr, err := accessToken.SignedString(s.jwtSecret)
	if err != nil {
		return nil, fmt.Errorf("failed to sign access token: %w", err)
	}

	// Refresh Token
	refreshTokenID := uuid.New().String()
	refreshClaims := RefreshClaims{
		TokenID: refreshTokenID,
		UserID:  user.ID,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   user.ID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(refreshExpiry),
			Issuer:    "pulseflow-auth",
		},
	}
	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
	refreshStr, err := refreshToken.SignedString(s.jwtSecret)
	if err != nil {
		return nil, fmt.Errorf("failed to sign refresh token: %w", err)
	}

	// Persist refresh token record for revocation tracking
	rec := &repository.RefreshTokenRecord{
		TokenID:   refreshTokenID,
		UserID:    user.ID,
		ExpiresAt: refreshExpiry,
		Revoked:   false,
	}
	if err := s.tokenStore.Save(ctx, rec); err != nil {
		return nil, fmt.Errorf("failed to persist refresh token: %w", err)
	}

	return &TokenPair{
		AccessToken:  accessStr,
		RefreshToken: refreshStr,
		ExpiresIn:    int(AccessTokenDuration.Seconds()),
	}, nil
}

// ValidateAccessToken parses and validates the access token
func (s *AuthService) ValidateAccessToken(tokenStr string) (*CustomClaims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &CustomClaims{}, func(t *jwt.Token) (interface{}, error) {
		// Strict algorithm check: prevent "none" algorithm attack!
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing algorithm: %v", t.Header["alg"])
		}
		return s.jwtSecret, nil
	})

	if err != nil || !token.Valid {
		return nil, errcode.NewUnauthorized("Invalid, expired, or tampered access token")
	}

	claims, ok := token.Claims.(*CustomClaims)
	if !ok {
		return nil, errcode.NewUnauthorized("Malformed token claims")
	}

	return claims, nil
}

// RefreshToken implements token rotation: validates old refresh token, revokes it, and issues a new pair
func (s *AuthService) RefreshToken(ctx context.Context, refreshTokenStr string) (*TokenPair, error) {
	token, err := jwt.ParseWithClaims(refreshTokenStr, &RefreshClaims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing algorithm: %v", t.Header["alg"])
		}
		return s.jwtSecret, nil
	})

	if err != nil || !token.Valid {
		return nil, errcode.NewUnauthorized("Invalid or expired refresh token")
	}

	claims, ok := token.Claims.(*RefreshClaims)
	if !ok {
		return nil, errcode.NewUnauthorized("Malformed refresh token claims")
	}

	// Check token store for revocation
	rec, err := s.tokenStore.Get(ctx, claims.TokenID)
	if err != nil {
		return nil, errcode.NewUnauthorized("Refresh token not found")
	}

	if rec.Revoked {
		// Security breach indicator: revoked token was reused!
		// Immediately revoke ALL tokens for this user for security
		_ = s.tokenStore.RevokeAllForUser(ctx, claims.UserID)
		return nil, errcode.NewUnauthorized("Refresh token reuse detected. All sessions revoked for security.")
	}

	// Revoke the old token (token rotation)
	_ = s.tokenStore.Revoke(ctx, claims.TokenID)

	// Fetch user to issue new pair
	user, err := s.userRepo.GetByID(ctx, claims.UserID)
	if err != nil || !user.IsActive {
		return nil, errcode.NewUnauthorized("User account inactive or deleted")
	}

	return s.GenerateTokenPair(ctx, user)
}

// Logout revokes refresh token
func (s *AuthService) Logout(ctx context.Context, refreshTokenStr string) error {
	token, err := jwt.ParseWithClaims(refreshTokenStr, &RefreshClaims{}, func(t *jwt.Token) (interface{}, error) {
		return s.jwtSecret, nil
	})
	if err == nil && token.Valid {
		if claims, ok := token.Claims.(*RefreshClaims); ok {
			return s.tokenStore.Revoke(ctx, claims.TokenID)
		}
	}
	return nil
}

package repository

import (
	"context"
	"sync"
	"time"

	"project-management-system/pkg/errcode"
)

type RefreshTokenRecord struct {
	TokenID   string
	UserID    string
	ExpiresAt time.Time
	Revoked   bool
}

type TokenStore interface {
	Save(ctx context.Context, record *RefreshTokenRecord) error
	Get(ctx context.Context, tokenID string) (*RefreshTokenRecord, error)
	Revoke(ctx context.Context, tokenID string) error
	RevokeAllForUser(ctx context.Context, userID string) error
}

type InMemoryTokenStore struct {
	mu     sync.RWMutex
	tokens map[string]*RefreshTokenRecord
}

func NewInMemoryTokenStore() *InMemoryTokenStore {
	return &InMemoryTokenStore{
		tokens: make(map[string]*RefreshTokenRecord),
	}
}

func (s *InMemoryTokenStore) Save(ctx context.Context, record *RefreshTokenRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tokens[record.TokenID] = record
	return nil
}

func (s *InMemoryTokenStore) Get(ctx context.Context, tokenID string) (*RefreshTokenRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rec, ok := s.tokens[tokenID]
	if !ok {
		return nil, errcode.NewUnauthorized("Invalid or expired refresh token")
	}
	r := *rec
	return &r, nil
}

func (s *InMemoryTokenStore) Revoke(ctx context.Context, tokenID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	rec, ok := s.tokens[tokenID]
	if ok {
		rec.Revoked = true
	}
	return nil
}

func (s *InMemoryTokenStore) RevokeAllForUser(ctx context.Context, userID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, rec := range s.tokens {
		if rec.UserID == userID {
			rec.Revoked = true
		}
	}
	return nil
}

package repository

import (
	"context"
	"strings"
	"sync"
	"time"

	"project-management-system/internal/domain"
	"project-management-system/pkg/errcode"
)

type UserRepository interface {
	Create(ctx context.Context, user *domain.User) error
	GetByID(ctx context.Context, id string) (*domain.User, error)
	GetByEmail(ctx context.Context, email string) (*domain.User, error)
	List(ctx context.Context, role domain.SystemRole) ([]*domain.User, error)
	Update(ctx context.Context, user *domain.User) error
}

type InMemoryUserRepository struct {
	mu    sync.RWMutex
	users map[string]*domain.User
}

func NewInMemoryUserRepository() *InMemoryUserRepository {
	repo := &InMemoryUserRepository{
		users: make(map[string]*domain.User),
	}
	// Seed initial admin user (password: AdminPass123!)
	// bcrypt hash for AdminPass123! with cost 12
	now := time.Now().UTC()
	adminUser := &domain.User{
		ID:                "usr-admin-01",
		Email:             "admin@pulseflow.io",
		PasswordHash:      "$2a$12$6k.pQp8Jp/5aJq8y3jD5aOU.hD8g0gqg0vIuL2H4U6wZ8lF1Z2k.y", // placeholder, will set via service in tests
		FullName:          "System Administrator",
		PrimaryRole:       domain.RoleAdmin,
		DefaultDailyHours: 8.0,
		IsActive:          true,
		CreatedAt:         now,
		UpdatedAt:         now,
	}
	repo.users[adminUser.ID] = adminUser
	return repo
}

func (r *InMemoryUserRepository) Create(ctx context.Context, user *domain.User) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	normEmail := strings.ToLower(strings.TrimSpace(user.Email))
	for _, u := range r.users {
		if strings.ToLower(u.Email) == normEmail {
			return errcode.NewConflict("A user with this email address already exists", map[string]string{"email": user.Email})
		}
	}

	r.users[user.ID] = user
	return nil
}

func (r *InMemoryUserRepository) GetByID(ctx context.Context, id string) (*domain.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	user, ok := r.users[id]
	if !ok {
		return nil, errcode.NewNotFound("User not found")
	}
	// Return shallow copy
	u := *user
	return &u, nil
}

func (r *InMemoryUserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	normEmail := strings.ToLower(strings.TrimSpace(email))
	for _, u := range r.users {
		if strings.ToLower(u.Email) == normEmail {
			userCopy := *u
			return &userCopy, nil
		}
	}
	return nil, errcode.NewNotFound("User not found")
}

func (r *InMemoryUserRepository) List(ctx context.Context, role domain.SystemRole) ([]*domain.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]*domain.User, 0, len(r.users))
	for _, u := range r.users {
		if role != "" && u.PrimaryRole != role {
			continue
		}
		userCopy := *u
		result = append(result, &userCopy)
	}
	return result, nil
}

func (r *InMemoryUserRepository) Update(ctx context.Context, user *domain.User) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	existing, ok := r.users[user.ID]
	if !ok {
		return errcode.NewNotFound("User not found")
	}

	// Email uniqueness check if email changed
	normEmail := strings.ToLower(strings.TrimSpace(user.Email))
	for id, u := range r.users {
		if id != user.ID && strings.ToLower(u.Email) == normEmail {
			return errcode.NewConflict("Email is already used by another user", map[string]string{"email": user.Email})
		}
	}

	user.UpdatedAt = time.Now().UTC()
	user.CreatedAt = existing.CreatedAt
	r.users[user.ID] = user
	return nil
}

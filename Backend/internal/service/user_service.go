package service

import (
	"context"
	"time"

	"github.com/google/uuid"

	"project-management-system/internal/domain"
	"project-management-system/internal/repository"
	"project-management-system/pkg/errcode"
)

type CreateUserDTO struct {
	Email             string            `json:"email"`
	Password          string            `json:"password"`
	FullName          string            `json:"full_name"`
	PrimaryRole       domain.SystemRole `json:"primary_role"`
	DefaultDailyHours float64           `json:"default_daily_hours"`
}

type UpdateUserDTO struct {
	FullName          *string            `json:"full_name,omitempty"`
	PrimaryRole       *domain.SystemRole `json:"primary_role,omitempty"`
	DefaultDailyHours *float64           `json:"default_daily_hours,omitempty"`
	IsActive          *bool              `json:"is_active,omitempty"`
}

type UserService struct {
	repo        repository.UserRepository
	authService *AuthService
}

func NewUserService(repo repository.UserRepository, authService *AuthService) *UserService {
	return &UserService{
		repo:        repo,
		authService: authService,
	}
}

func (s *UserService) Create(ctx context.Context, dto CreateUserDTO) (*domain.User, error) {
	if len(dto.Password) < 8 {
		return nil, errcode.NewValidation("Password must be at least 8 characters long", nil)
	}

	hash, err := s.authService.HashPassword(dto.Password)
	if err != nil {
		return nil, errcode.NewInternal("Failed to process password")
	}

	dailyHours := dto.DefaultDailyHours
	if dailyHours <= 0 {
		dailyHours = 8.0
	}

	now := time.Now().UTC()
	user := &domain.User{
		ID:                uuid.New().String(),
		Email:             dto.Email,
		PasswordHash:      hash,
		FullName:          dto.FullName,
		PrimaryRole:       dto.PrimaryRole,
		DefaultDailyHours: dailyHours,
		IsActive:          true,
		CreatedAt:         now,
		UpdatedAt:         now,
	}

	if err := user.Validate(); err != nil {
		return nil, err
	}

	if err := s.repo.Create(ctx, user); err != nil {
		return nil, err
	}

	return user, nil
}

func (s *UserService) GetByID(ctx context.Context, id string) (*domain.User, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *UserService) List(ctx context.Context, role domain.SystemRole) ([]*domain.User, error) {
	return s.repo.List(ctx, role)
}

func (s *UserService) Update(ctx context.Context, id string, dto UpdateUserDTO) (*domain.User, error) {
	user, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if dto.FullName != nil {
		user.FullName = *dto.FullName
	}
	if dto.PrimaryRole != nil {
		user.PrimaryRole = *dto.PrimaryRole
	}
	if dto.DefaultDailyHours != nil {
		user.DefaultDailyHours = *dto.DefaultDailyHours
	}
	if dto.IsActive != nil {
		user.IsActive = *dto.IsActive
	}

	if err := user.Validate(); err != nil {
		return nil, err
	}

	if err := s.repo.Update(ctx, user); err != nil {
		return nil, err
	}

	return user, nil
}

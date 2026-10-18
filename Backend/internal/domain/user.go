package domain

import (
	"fmt"
	"strings"
	"time"

	"project-management-system/pkg/errcode"
)

type SystemRole string

const (
	RoleDeveloper       SystemRole = "DEVELOPER"
	RoleQAEngineer      SystemRole = "QA_ENGINEER"
	RoleBusinessAnalyst SystemRole = "BUSINESS_ANALYST"
	RoleProjectManager  SystemRole = "PROJECT_MANAGER"
	RoleAdmin           SystemRole = "ADMIN"
)

func (r SystemRole) IsValid() bool {
	switch r {
	case RoleDeveloper, RoleQAEngineer, RoleBusinessAnalyst, RoleProjectManager, RoleAdmin:
		return true
	default:
		return false
	}
}

type User struct {
	ID                string     `json:"id"`
	Email             string     `json:"email"`
	PasswordHash      string     `json:"-"`
	FullName          string     `json:"full_name"`
	PrimaryRole       SystemRole `json:"primary_role"`
	DefaultDailyHours float64    `json:"default_daily_hours"`
	IsActive          bool       `json:"is_active"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

func (u *User) Validate() error {
	if strings.TrimSpace(u.Email) == "" || !strings.Contains(u.Email, "@") {
		return errcode.NewValidation("A valid email address is required", nil)
	}
	if strings.TrimSpace(u.FullName) == "" {
		return errcode.NewValidation("Full name cannot be blank", nil)
	}
	if !u.PrimaryRole.IsValid() {
		return errcode.NewValidation(fmt.Sprintf("Invalid primary role '%s'", u.PrimaryRole), nil)
	}
	if u.DefaultDailyHours <= 0 || u.DefaultDailyHours > 24 {
		return errcode.NewValidation("Default daily working hours must be between 0.1 and 24.0", nil)
	}
	return nil
}

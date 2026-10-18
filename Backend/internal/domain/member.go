package domain

import (
	"time"

	"project-management-system/pkg/errcode"
)

type ProjectRole string

const (
	ProjectRoleDeveloper ProjectRole = "DEVELOPER"
	ProjectRoleQA        ProjectRole = "QA"
	ProjectRoleBA        ProjectRole = "BA"
	ProjectRolePM        ProjectRole = "PM"
)

func (r ProjectRole) IsValid() bool {
	switch r {
	case ProjectRoleDeveloper, ProjectRoleQA, ProjectRoleBA, ProjectRolePM:
		return true
	default:
		return false
	}
}

type ProjectMember struct {
	ID                  string      `json:"id"`
	ProjectID           string      `json:"project_id"`
	UserID              string      `json:"user_id"`
	ProjectRole         ProjectRole `json:"project_role"`
	AllocationPct       int         `json:"allocation_pct"`
	EffectiveDailyHours float64     `json:"effective_daily_hours"`
	JoinedDate          time.Time   `json:"joined_date"`
	LeftDate            *time.Time  `json:"left_date,omitempty"`
}

func (m *ProjectMember) Validate() error {
	if !m.ProjectRole.IsValid() {
		return errcode.NewValidation("Invalid project role. Allowed: DEVELOPER, QA, BA, PM", nil)
	}
	if m.AllocationPct < 1 || m.AllocationPct > 100 {
		return errcode.NewValidation("Allocation percentage must be between 1 and 100", nil)
	}
	if m.EffectiveDailyHours <= 0 || m.EffectiveDailyHours > 24 {
		return errcode.NewValidation("Effective daily hours must be between 0.1 and 24.0", nil)
	}
	if m.LeftDate != nil && m.LeftDate.Before(m.JoinedDate) {
		return errcode.NewValidation("Member left date cannot be earlier than joined date", nil)
	}
	return nil
}

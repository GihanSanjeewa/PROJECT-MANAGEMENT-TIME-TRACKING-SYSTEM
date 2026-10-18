package domain

import (
	"strings"
	"time"

	"project-management-system/pkg/errcode"
)

type ProjectStatus string

const (
	ProjectStatusDraft     ProjectStatus = "DRAFT"
	ProjectStatusActive    ProjectStatus = "ACTIVE"
	ProjectStatusOnHold    ProjectStatus = "ON_HOLD"
	ProjectStatusCompleted ProjectStatus = "COMPLETED"
)

func (s ProjectStatus) IsValid() bool {
	switch s {
	case ProjectStatusDraft, ProjectStatusActive, ProjectStatusOnHold, ProjectStatusCompleted:
		return true
	default:
		return false
	}
}

type Project struct {
	ID            string        `json:"id"`
	Code          string        `json:"code"`
	Name          string        `json:"name"`
	Description   string        `json:"description"`
	StartDate     time.Time     `json:"start_date"`
	TargetEndDate time.Time     `json:"target_end_date"`
	BudgetHours   float64       `json:"budget_hours"`
	Status        ProjectStatus `json:"status"`
	CreatedBy     string        `json:"created_by"`
	CreatedAt     time.Time     `json:"created_at"`
	UpdatedAt     time.Time     `json:"updated_at"`
}

func (p *Project) Validate() error {
	if strings.TrimSpace(p.Code) == "" {
		return errcode.NewValidation("Project code cannot be blank", nil)
	}
	if strings.TrimSpace(p.Name) == "" {
		return errcode.NewValidation("Project name cannot be blank", nil)
	}
	if p.TargetEndDate.Before(p.StartDate) {
		return errcode.NewValidation("Target end date must be on or after start date", nil)
	}
	if p.BudgetHours < 0 {
		return errcode.NewValidation("Budget hours cannot be negative", nil)
	}
	if p.Status != "" && !p.Status.IsValid() {
		return errcode.NewValidation("Invalid project status", nil)
	}
	return nil
}

// ContainsDate checks if a given date falls within the project timeline boundaries
func (p *Project) ContainsDate(d time.Time) bool {
	dDay := time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, time.UTC)
	startDay := time.Date(p.StartDate.Year(), p.StartDate.Month(), p.StartDate.Day(), 0, 0, 0, 0, time.UTC)
	endDay := time.Date(p.TargetEndDate.Year(), p.TargetEndDate.Month(), p.TargetEndDate.Day(), 0, 0, 0, 0, time.UTC)

	return !dDay.Before(startDay) && !dDay.After(endDay)
}

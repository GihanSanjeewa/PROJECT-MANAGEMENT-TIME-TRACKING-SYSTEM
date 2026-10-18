package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"project-management-system/internal/domain"
	"project-management-system/internal/repository"
	"project-management-system/pkg/errcode"
)

type CreateProjectDTO struct {
	Code          string    `json:"code"`
	Name          string    `json:"name"`
	Description   string    `json:"description"`
	StartDate     time.Time `json:"start_date"`
	TargetEndDate time.Time `json:"target_end_date"`
	BudgetHours   float64   `json:"budget_hours"`
	CreatedBy     string    `json:"created_by"`
}

type UpdateProjectDTO struct {
	Name          *string               `json:"name,omitempty"`
	Description   *string               `json:"description,omitempty"`
	StartDate     *time.Time            `json:"start_date,omitempty"`
	TargetEndDate *time.Time            `json:"target_end_date,omitempty"`
	BudgetHours   *float64              `json:"budget_hours,omitempty"`
	Status        *domain.ProjectStatus `json:"status,omitempty"`
}

type AssignMemberDTO struct {
	ProjectID           string             `json:"project_id"`
	UserID              string             `json:"user_id"`
	ProjectRole         domain.ProjectRole `json:"project_role"`
	AllocationPct       int                `json:"allocation_pct"`
	EffectiveDailyHours float64            `json:"effective_daily_hours"`
	JoinedDate          time.Time          `json:"joined_date"`
	LeftDate            *time.Time         `json:"left_date,omitempty"`
}

type RoleCapacityDTO struct {
	Role               domain.ProjectRole `json:"role"`
	MemberCount        int                `json:"member_count"`
	TotalCapacityHours float64            `json:"total_capacity_hours"`
	WorkingDays        int                `json:"working_days"`
}

type CapacityResponseDTO struct {
	ProjectID           string                     `json:"project_id"`
	ProjectCode         string                     `json:"project_code"`
	ProjectName         string                     `json:"project_name"`
	StartDate           time.Time                  `json:"start_date"`
	TargetEndDate       time.Time                  `json:"target_end_date"`
	TotalWorkingDays    int                        `json:"total_working_days"`
	TotalCapacityHours  float64                    `json:"total_capacity_hours"`
	Roles               map[string]RoleCapacityDTO `json:"roles"`
}

type ProjectService struct {
	repo         repository.ProjectRepository
	userRepo     repository.UserRepository
	calendarServ *CalendarService
}

func NewProjectService(
	repo repository.ProjectRepository,
	userRepo repository.UserRepository,
	calendarServ *CalendarService,
) *ProjectService {
	return &ProjectService{
		repo:         repo,
		userRepo:     userRepo,
		calendarServ: calendarServ,
	}
}

func (s *ProjectService) Create(ctx context.Context, dto CreateProjectDTO) (*domain.Project, error) {
	now := time.Now().UTC()
	project := &domain.Project{
		ID:            uuid.New().String(),
		Code:          dto.Code,
		Name:          dto.Name,
		Description:   dto.Description,
		StartDate:     dto.StartDate,
		TargetEndDate: dto.TargetEndDate,
		BudgetHours:   dto.BudgetHours,
		Status:        domain.ProjectStatusActive,
		CreatedBy:     dto.CreatedBy,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	if err := project.Validate(); err != nil {
		return nil, err
	}

	if err := s.repo.Create(ctx, project); err != nil {
		return nil, err
	}

	return project, nil
}

func (s *ProjectService) GetByID(ctx context.Context, id string) (*domain.Project, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *ProjectService) List(ctx context.Context) ([]*domain.Project, error) {
	return s.repo.List(ctx)
}

// Update handles project updates with zero-bug timeline shrinking guard (TSK-014 / QA-02)
func (s *ProjectService) Update(ctx context.Context, id string, dto UpdateProjectDTO) (*domain.Project, error) {
	project, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	newStart := project.StartDate
	newEnd := project.TargetEndDate

	if dto.StartDate != nil {
		newStart = *dto.StartDate
	}
	if dto.TargetEndDate != nil {
		newEnd = *dto.TargetEndDate
	}

	if newEnd.Before(newStart) {
		return nil, errcode.NewValidation("Target end date cannot be earlier than start date", nil)
	}

	// TIMELINE SHRINKING GUARD: Verify existing tasks do not exceed the new bounds
	minTaskStart, maxTaskDue, taskIDs, err := s.repo.GetTaskDateBounds(ctx, id)
	if err == nil && len(taskIDs) > 0 {
		if minTaskStart != nil && newStart.After(*minTaskStart) {
			return nil, errcode.NewConflict(
				fmt.Sprintf("Cannot shrink project start date to %s. Existing task starts earlier on %s",
					newStart.Format("2006-01-02"), minTaskStart.Format("2006-01-02")),
				map[string]interface{}{
					"new_start_date":   newStart.Format("2006-01-02"),
					"earliest_task":    minTaskStart.Format("2006-01-02"),
					"conflicting_tasks": taskIDs,
				},
			)
		}

		if maxTaskDue != nil && newEnd.Before(*maxTaskDue) {
			return nil, errcode.NewConflict(
				fmt.Sprintf("Cannot shrink project target end date to %s. Existing task due date extends to %s",
					newEnd.Format("2006-01-02"), maxTaskDue.Format("2006-01-02")),
				map[string]interface{}{
					"new_target_end_date": newEnd.Format("2006-01-02"),
					"latest_task_due":     maxTaskDue.Format("2006-01-02"),
					"conflicting_tasks":    taskIDs,
				},
			)
		}
	}

	if dto.Name != nil {
		project.Name = *dto.Name
	}
	if dto.Description != nil {
		project.Description = *dto.Description
	}
	if dto.BudgetHours != nil {
		project.BudgetHours = *dto.BudgetHours
	}
	if dto.Status != nil {
		project.Status = *dto.Status
	}
	project.StartDate = newStart
	project.TargetEndDate = newEnd

	if err := project.Validate(); err != nil {
		return nil, err
	}

	if err := s.repo.Update(ctx, project); err != nil {
		return nil, err
	}

	return project, nil
}

// AddMember assigns a user to the project team with functional role & allocation
func (s *ProjectService) AddMember(ctx context.Context, dto AssignMemberDTO) (*domain.ProjectMember, error) {
	// 1. Verify project exists
	project, err := s.repo.GetByID(ctx, dto.ProjectID)
	if err != nil {
		return nil, errcode.NewNotFound("Project not found")
	}

	// 2. Verify user exists
	user, err := s.userRepo.GetByID(ctx, dto.UserID)
	if err != nil {
		return nil, errcode.NewNotFound("User not found")
	}

	dailyHours := dto.EffectiveDailyHours
	if dailyHours <= 0 {
		dailyHours = user.DefaultDailyHours
	}

	joinedDate := dto.JoinedDate
	if joinedDate.IsZero() {
		joinedDate = project.StartDate
	}

	member := &domain.ProjectMember{
		ID:                  uuid.New().String(),
		ProjectID:           dto.ProjectID,
		UserID:              dto.UserID,
		ProjectRole:         dto.ProjectRole,
		AllocationPct:       dto.AllocationPct,
		EffectiveDailyHours: dailyHours,
		JoinedDate:          joinedDate,
		LeftDate:            dto.LeftDate,
	}

	if member.AllocationPct <= 0 {
		member.AllocationPct = 100
	}

	if err := member.Validate(); err != nil {
		return nil, err
	}

	if err := s.repo.AddMember(ctx, member); err != nil {
		return nil, err
	}

	return member, nil
}

func (s *ProjectService) GetMembers(ctx context.Context, projectID string) ([]*domain.ProjectMember, error) {
	return s.repo.GetMembers(ctx, projectID)
}

func (s *ProjectService) RemoveMember(ctx context.Context, projectID, userID string) error {
	return s.repo.RemoveMember(ctx, projectID, userID)
}

// CalculateTeamCapacity computes total working capacity hours by role (TSK-016)
func (s *ProjectService) CalculateTeamCapacity(ctx context.Context, projectID string) (*CapacityResponseDTO, error) {
	project, err := s.repo.GetByID(ctx, projectID)
	if err != nil {
		return nil, err
	}

	members, err := s.repo.GetMembers(ctx, projectID)
	if err != nil {
		return nil, err
	}

	holidays, err := s.repo.GetHolidays(ctx)
	if err != nil {
		return nil, err
	}

	totalWorkingDays := s.calendarServ.CalculateWorkingDays(project.StartDate, project.TargetEndDate, holidays)

	rolesMap := map[string]RoleCapacityDTO{
		string(domain.ProjectRoleDeveloper): {Role: domain.ProjectRoleDeveloper, WorkingDays: totalWorkingDays},
		string(domain.ProjectRoleQA):        {Role: domain.ProjectRoleQA, WorkingDays: totalWorkingDays},
		string(domain.ProjectRoleBA):        {Role: domain.ProjectRoleBA, WorkingDays: totalWorkingDays},
		string(domain.ProjectRolePM):        {Role: domain.ProjectRolePM, WorkingDays: totalWorkingDays},
	}

	var totalProjectCapacity float64 = 0

	for _, m := range members {
		// Effective working window for this member
		memberStart := m.JoinedDate
		if memberStart.Before(project.StartDate) {
			memberStart = project.StartDate
		}

		memberEnd := project.TargetEndDate
		if m.LeftDate != nil && m.LeftDate.Before(project.TargetEndDate) {
			memberEnd = *m.LeftDate
		}

		memberDays := s.calendarServ.CalculateWorkingDays(memberStart, memberEnd, holidays)
		memberCapacity := float64(memberDays) * m.EffectiveDailyHours * (float64(m.AllocationPct) / 100.0)

		roleKey := string(m.ProjectRole)
		current := rolesMap[roleKey]
		current.MemberCount++
		current.TotalCapacityHours += memberCapacity
		rolesMap[roleKey] = current

		totalProjectCapacity += memberCapacity
	}

	return &CapacityResponseDTO{
		ProjectID:          project.ID,
		ProjectCode:        project.Code,
		ProjectName:        project.Name,
		StartDate:          project.StartDate,
		TargetEndDate:      project.TargetEndDate,
		TotalWorkingDays:   totalWorkingDays,
		TotalCapacityHours: totalProjectCapacity,
		Roles:              rolesMap,
	}, nil
}

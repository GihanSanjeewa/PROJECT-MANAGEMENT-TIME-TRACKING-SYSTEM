package service_test

import (
	"context"
	"testing"
	"time"

	"project-management-system/internal/domain"
	"project-management-system/internal/repository"
	"project-management-system/internal/service"
	"project-management-system/pkg/errcode"
)

func setupProjectTest() (*service.ProjectService, *repository.InMemoryProjectRepository, *service.UserService) {
	userRepo := repository.NewInMemoryUserRepository()
	tokenStore := repository.NewInMemoryTokenStore()
	projectRepo := repository.NewInMemoryProjectRepository()

	calendarServ := service.NewCalendarService()
	authService := service.NewAuthService(userRepo, tokenStore, "super-secure-test-jwt-secret-with-at-least-32-chars!")
	userService := service.NewUserService(userRepo, authService)
	projectService := service.NewProjectService(projectRepo, userRepo, calendarServ)

	return projectService, projectRepo, userService
}

func TestProjectTimelineShrinkConflict(t *testing.T) {
	projectService, projectRepo, _ := setupProjectTest()
	ctx := context.Background()

	// 1. Create Project from Nov 01 to Nov 30
	startDate := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
	targetEnd := time.Date(2026, 11, 30, 0, 0, 0, 0, time.UTC)

	proj, err := projectService.Create(ctx, service.CreateProjectDTO{
		Code:          "TEST-01",
		Name:          "Shrink Guard Test Project",
		StartDate:     startDate,
		TargetEndDate: targetEnd,
	})
	if err != nil {
		t.Fatalf("failed to create project: %v", err)
	}

	// 2. Register an existing task ending on Nov 25
	taskDue := time.Date(2026, 11, 25, 0, 0, 0, 0, time.UTC)
	projectRepo.SetTaskForGuard(&domain.Task{
		ID:        "task-conflict-1",
		ProjectID: proj.ID,
		Title:     "Critical QA Task",
		StartDate: time.Date(2026, 11, 10, 0, 0, 0, 0, time.UTC),
		DueDate:   taskDue,
	})

	// 3. Attempt to shrink project target_end_date to Nov 20 (< Nov 25)
	shrunkEnd := time.Date(2026, 11, 20, 0, 0, 0, 0, time.UTC)
	_, err = projectService.Update(ctx, proj.ID, service.UpdateProjectDTO{
		TargetEndDate: &shrunkEnd,
	})

	if err == nil {
		t.Fatalf("ZERO-BUG GATE FAILED: Expected 409 Conflict when shrinking project deadline past existing task, got nil")
	}

	appErr, ok := err.(*errcode.AppError)
	if !ok || appErr.Code != errcode.ErrConflict {
		t.Errorf("expected ErrConflict (409), got: %v", err)
	}
}

func TestWorkingDaysCalculation(t *testing.T) {
	calendarServ := service.NewCalendarService()

	// Monday Nov 2 to Friday Nov 6 -> exactly 5 days (Mon, Tue, Wed, Thu, Fri)
	start := time.Date(2026, 11, 2, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 11, 6, 0, 0, 0, 0, time.UTC)

	holidays := []*domain.CompanyHoliday{
		{
			HolidayDate: time.Date(2026, 11, 4, 0, 0, 0, 0, time.UTC), // Wednesday holiday
			Name:        "Mid-week Holiday",
		},
	}

	// 5 days minus 1 holiday = 4 working days
	days := calendarServ.CalculateWorkingDays(start, end, holidays)
	if days != 4 {
		t.Errorf("expected 4 working days, got %d", days)
	}

	// Range covering full weekend (Friday to Monday: Fri, Sat, Sun, Mon) -> exactly 2 working days
	fri := time.Date(2026, 11, 6, 0, 0, 0, 0, time.UTC)
	mon := time.Date(2026, 11, 9, 0, 0, 0, 0, time.UTC)
	daysWeekend := calendarServ.CalculateWorkingDays(fri, mon, nil)
	if daysWeekend != 2 {
		t.Errorf("expected 2 working days (excluding Sat and Sun), got %d", daysWeekend)
	}
}

func TestTeamCapacityCalculation(t *testing.T) {
	projectService, _, userService := setupProjectTest()
	ctx := context.Background()

	// Create users
	dev, _ := userService.Create(ctx, service.CreateUserDTO{
		Email:       "dev1@pulseflow.io",
		Password:    "SecurePass123!",
		FullName:    "Dev One",
		PrimaryRole: domain.RoleDeveloper,
	})
	ba, _ := userService.Create(ctx, service.CreateUserDTO{
		Email:       "ba1@pulseflow.io",
		Password:    "SecurePass123!",
		FullName:    "BA One",
		PrimaryRole: domain.RoleBusinessAnalyst,
	})

	// Project running Monday Nov 2 to Friday Nov 6 (5 business days)
	start := time.Date(2026, 11, 2, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 11, 6, 0, 0, 0, 0, time.UTC)

	proj, _ := projectService.Create(ctx, service.CreateProjectDTO{
		Code:          "CAP-01",
		Name:          "Capacity Project",
		StartDate:     start,
		TargetEndDate: end,
	})

	// Assign Developer: 100% allocation, 8.0 hrs/day -> 5 days * 8h = 40h
	_, _ = projectService.AddMember(ctx, service.AssignMemberDTO{
		ProjectID:           proj.ID,
		UserID:              dev.ID,
		ProjectRole:         domain.ProjectRoleDeveloper,
		AllocationPct:       100,
		EffectiveDailyHours: 8.0,
		JoinedDate:          start,
	})

	// Assign BA: 50% allocation, 4.0 hrs/day -> 5 days * 4h * 50% = 10h
	_, _ = projectService.AddMember(ctx, service.AssignMemberDTO{
		ProjectID:           proj.ID,
		UserID:              ba.ID,
		ProjectRole:         domain.ProjectRoleBA,
		AllocationPct:       50,
		EffectiveDailyHours: 4.0,
		JoinedDate:          start,
	})

	capReport, err := projectService.CalculateTeamCapacity(ctx, proj.ID)
	if err != nil {
		t.Fatalf("failed to calculate capacity: %v", err)
	}

	devCap := capReport.Roles[string(domain.ProjectRoleDeveloper)].TotalCapacityHours
	if devCap != 40.0 {
		t.Errorf("expected 40.0 developer capacity hours, got %.2f", devCap)
	}

	baCap := capReport.Roles[string(domain.ProjectRoleBA)].TotalCapacityHours
	if baCap != 10.0 {
		t.Errorf("expected 10.0 BA capacity hours, got %.2f", baCap)
	}
}

func TestDuplicateRoleAssignmentRejection(t *testing.T) {
	projectService, _, userService := setupProjectTest()
	ctx := context.Background()

	dev, _ := userService.Create(ctx, service.CreateUserDTO{
		Email:       "dup@pulseflow.io",
		Password:    "SecurePass123!",
		FullName:    "Dup Dev",
		PrimaryRole: domain.RoleDeveloper,
	})

	start := time.Date(2026, 11, 2, 0, 0, 0, 0, time.UTC)
	proj, _ := projectService.Create(ctx, service.CreateProjectDTO{
		Code:          "DUP-01",
		Name:          "Dup Project",
		StartDate:     start,
		TargetEndDate: start.AddDate(0, 1, 0),
	})

	// First assignment succeeds
	_, err := projectService.AddMember(ctx, service.AssignMemberDTO{
		ProjectID:   proj.ID,
		UserID:      dev.ID,
		ProjectRole: domain.ProjectRoleDeveloper,
	})
	if err != nil {
		t.Fatalf("first member assignment should succeed: %v", err)
	}

	// Second duplicate assignment MUST fail with 409 Conflict
	_, err = projectService.AddMember(ctx, service.AssignMemberDTO{
		ProjectID:   proj.ID,
		UserID:      dev.ID,
		ProjectRole: domain.ProjectRoleDeveloper,
	})
	if err == nil {
		t.Fatalf("expected duplicate member assignment to be rejected with conflict, got nil")
	}
}

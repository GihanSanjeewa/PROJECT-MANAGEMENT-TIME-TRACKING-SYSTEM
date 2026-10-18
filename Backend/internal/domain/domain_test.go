package domain_test

import (
	"testing"
	"time"

	"project-management-system/internal/domain"
)

func TestProjectValidation_InvalidDates(t *testing.T) {
	proj := &domain.Project{
		Code:          "PRJ-01",
		Name:          "Banking Platform",
		StartDate:     time.Date(2026, 12, 1, 0, 0, 0, 0, time.UTC),
		TargetEndDate: time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC), // Target end before start!
		BudgetHours:   100,
	}

	err := proj.Validate()
	if err == nil {
		t.Fatalf("expected validation error when target_end_date < start_date, got nil")
	}
}

func TestTaskTimelineInvariant(t *testing.T) {
	proj := &domain.Project{
		Code:          "PRJ-01",
		Name:          "Banking Platform",
		StartDate:     time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC),
		TargetEndDate: time.Date(2026, 11, 30, 0, 0, 0, 0, time.UTC),
	}

	// 1. Task starting before project start
	taskEarly := &domain.Task{
		Title:          "Early Task",
		RequiredRole:   domain.RoleRequiredDev,
		EstimatedHours: 8,
		StartDate:      time.Date(2026, 10, 31, 0, 0, 0, 0, time.UTC), // 1 day before project start
		DueDate:        time.Date(2026, 11, 5, 0, 0, 0, 0, time.UTC),
	}
	if err := taskEarly.AssertWithinProjectTimeline(proj); err == nil {
		t.Fatalf("expected error when task starts before project start date, got nil")
	}

	// 2. Task due after project target end
	taskLate := &domain.Task{
		Title:          "Late Task",
		RequiredRole:   domain.RoleRequiredQA,
		EstimatedHours: 8,
		StartDate:      time.Date(2026, 11, 10, 0, 0, 0, 0, time.UTC),
		DueDate:        time.Date(2026, 12, 1, 0, 0, 0, 0, time.UTC), // 1 day after project target end
	}
	if err := taskLate.AssertWithinProjectTimeline(proj); err == nil {
		t.Fatalf("expected error when task due date exceeds project target end date, got nil")
	}

	// 3. Valid task within project timeline
	taskValid := &domain.Task{
		Title:          "Valid Task",
		RequiredRole:   domain.RoleRequiredDev,
		EstimatedHours: 16,
		StartDate:      time.Date(2026, 11, 5, 0, 0, 0, 0, time.UTC),
		DueDate:        time.Date(2026, 11, 20, 0, 0, 0, 0, time.UTC),
	}
	if err := taskValid.AssertWithinProjectTimeline(proj); err != nil {
		t.Fatalf("expected valid task to pass timeline assertion, got error: %v", err)
	}
}

func TestTaskRemainingHoursCalculation(t *testing.T) {
	task := &domain.Task{
		Title:          "Feature Implementation",
		Status:         domain.TaskStatusInProgress,
		EstimatedHours: 20.0,
		LoggedHours:    7.5,
	}

	rem := task.RemainingHours()
	if rem != 12.5 {
		t.Errorf("expected remaining hours to be 12.5, got %.2f", rem)
	}

	// When logged hours exceed estimate, remaining hours should be 0 (not negative)
	task.LoggedHours = 25.0
	if remExceeded := task.RemainingHours(); remExceeded != 0 {
		t.Errorf("expected remaining hours to be 0 when logged > estimated, got %.2f", remExceeded)
	}

	// When task is marked Done, remaining hours must be 0
	task.Status = domain.TaskStatusDone
	task.LoggedHours = 10.0
	if remDone := task.RemainingHours(); remDone != 0 {
		t.Errorf("expected remaining hours to be 0 for completed task, got %.2f", remDone)
	}
}

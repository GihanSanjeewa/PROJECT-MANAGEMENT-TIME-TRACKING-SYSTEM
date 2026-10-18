package service

import (
	"context"
	"testing"
	"time"

	"project-management-system/internal/domain"
	"project-management-system/internal/repository"
)

func setupTestTaskService(t *testing.T) (*TaskService, *repository.InMemoryProjectRepository, *repository.InMemoryTaskRepository) {
	ctx := context.Background()
	userRepo := repository.NewInMemoryUserRepository()
	projectRepo := repository.NewInMemoryProjectRepository()
	taskRepo := repository.NewInMemoryTaskRepository(projectRepo)
	calendarService := NewCalendarService()
	taskService := NewTaskService(taskRepo, projectRepo, userRepo, calendarService)

	// Seed test project (2026-11-01 to 2026-12-31)
	proj := &domain.Project{
		ID:            "proj-test-1",
		Code:          "TST-01",
		Name:          "Test Core Project",
		StartDate:     time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC),
		TargetEndDate: time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC),
		Status:        domain.ProjectStatusActive,
		BudgetHours:   500,
	}
	if err := projectRepo.Create(ctx, proj); err != nil {
		t.Fatalf("Failed to seed project: %v", err)
	}

	// Seed users
	userRepo.Create(ctx, &domain.User{ID: "usr-dev", FullName: "Jordan Dev", Email: "jordan@pulseflow.io", PrimaryRole: domain.RoleDeveloper, DefaultDailyHours: 8.0, IsActive: true})
	userRepo.Create(ctx, &domain.User{ID: "usr-qa", FullName: "Maya QA", Email: "maya@pulseflow.io", PrimaryRole: domain.RoleQAEngineer, DefaultDailyHours: 8.0, IsActive: true})

	// Seed members in project
	projectRepo.AddMember(ctx, &domain.ProjectMember{
		ID:                  "m-dev",
		ProjectID:           "proj-test-1",
		UserID:              "usr-dev",
		ProjectRole:         domain.ProjectRoleDeveloper,
		AllocationPct:       100,
		EffectiveDailyHours: 8.0,
		JoinedDate:          time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC),
	})
	projectRepo.AddMember(ctx, &domain.ProjectMember{
		ID:                  "m-qa",
		ProjectID:           "proj-test-1",
		UserID:              "usr-qa",
		ProjectRole:         domain.ProjectRoleQA,
		AllocationPct:       100,
		EffectiveDailyHours: 8.0,
		JoinedDate:          time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC),
	})

	return taskService, projectRepo, taskRepo
}

// STEP-4.1 Quality Gate: Task Creation with Project Range Check
func TestTaskCreationTimelineBoundaries(t *testing.T) {
	svc, _, _ := setupTestTaskService(t)
	ctx := context.Background()

	devID := "usr-dev"

	tests := []struct {
		name        string
		task        *domain.Task
		shouldError bool
		errContains string
	}{
		{
			name: "Valid Task within Timeline",
			task: &domain.Task{
				ProjectID:      "proj-test-1",
				Title:          "Build Auth Middleware",
				RequiredRole:   domain.RoleRequiredDev,
				EstimatedHours: 16.0,
				StartDate:      time.Date(2026, 11, 5, 0, 0, 0, 0, time.UTC),
				DueDate:        time.Date(2026, 11, 10, 0, 0, 0, 0, time.UTC),
				AssignedTo:     &devID,
			},
			shouldError: false,
		},
		{
			name: "Task Starting Before Project Start Date",
			task: &domain.Task{
				ProjectID:      "proj-test-1",
				Title:          "Premature Task",
				RequiredRole:   domain.RoleRequiredDev,
				EstimatedHours: 8.0,
				StartDate:      time.Date(2026, 10, 31, 0, 0, 0, 0, time.UTC), // 1 day before Nov 1
				DueDate:        time.Date(2026, 11, 5, 0, 0, 0, 0, time.UTC),
				AssignedTo:     &devID,
			},
			shouldError: true,
			errContains: "cannot be earlier than project start date",
		},
		{
			name: "Task Ending After Project Target End Date",
			task: &domain.Task{
				ProjectID:      "proj-test-1",
				Title:          "Late Task",
				RequiredRole:   domain.RoleRequiredDev,
				EstimatedHours: 8.0,
				StartDate:      time.Date(2026, 12, 20, 0, 0, 0, 0, time.UTC),
				DueDate:        time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC), // 1 day after Dec 31
				AssignedTo:     &devID,
			},
			shouldError: true,
			errContains: "cannot exceed project target end date",
		},
		{
			name: "Task Assigned to Non-Project Member",
			task: &domain.Task{
				ProjectID:      "proj-test-1",
				Title:          "Ghost Assigned Task",
				RequiredRole:   domain.RoleRequiredDev,
				EstimatedHours: 8.0,
				StartDate:      time.Date(2026, 11, 10, 0, 0, 0, 0, time.UTC),
				DueDate:        time.Date(2026, 11, 15, 0, 0, 0, 0, time.UTC),
				AssignedTo:     func() *string { s := "usr-unknown"; return &s }(),
			},
			shouldError: true,
			errContains: "is not an assigned team member",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := svc.CreateTask(ctx, tc.task)
			if tc.shouldError {
				if err == nil {
					t.Fatalf("Expected error for %s, got nil", tc.name)
				}
			} else {
				if err != nil {
					t.Fatalf("Unexpected error for %s: %v", tc.name, err)
				}
			}
		})
	}
}

// STEP-4.2 Quality Gate: DAG Cycle Detection Algorithm
func TestTaskDependencyDAGCycleDetection(t *testing.T) {
	svc, _, _ := setupTestTaskService(t)
	ctx := context.Background()

	devID := "usr-dev"

	// Create 3 tasks: A -> B -> C
	taskA, err := svc.CreateTask(ctx, &domain.Task{
		ID:             "task-A",
		ProjectID:      "proj-test-1",
		Title:          "Task A",
		RequiredRole:   domain.RoleRequiredDev,
		EstimatedHours: 8.0,
		StartDate:      time.Date(2026, 11, 2, 0, 0, 0, 0, time.UTC),
		DueDate:        time.Date(2026, 11, 4, 0, 0, 0, 0, time.UTC),
		AssignedTo:     &devID,
	})
	if err != nil {
		t.Fatalf("Failed to create task A: %v", err)
	}

	taskB, err := svc.CreateTask(ctx, &domain.Task{
		ID:             "task-B",
		ProjectID:      "proj-test-1",
		Title:          "Task B",
		RequiredRole:   domain.RoleRequiredDev,
		EstimatedHours: 8.0,
		StartDate:      time.Date(2026, 11, 5, 0, 0, 0, 0, time.UTC),
		DueDate:        time.Date(2026, 11, 8, 0, 0, 0, 0, time.UTC),
		AssignedTo:     &devID,
	})
	if err != nil {
		t.Fatalf("Failed to create task B: %v", err)
	}

	taskC, err := svc.CreateTask(ctx, &domain.Task{
		ID:             "task-C",
		ProjectID:      "proj-test-1",
		Title:          "Task C",
		RequiredRole:   domain.RoleRequiredDev,
		EstimatedHours: 8.0,
		StartDate:      time.Date(2026, 11, 9, 0, 0, 0, 0, time.UTC),
		DueDate:        time.Date(2026, 11, 12, 0, 0, 0, 0, time.UTC),
		AssignedTo:     &devID,
	})
	if err != nil {
		t.Fatalf("Failed to create task C: %v", err)
	}

	// 1. Add valid dependency: Task B depends on Task A (A -> B)
	_, err = svc.AddDependency(ctx, "proj-test-1", taskB.ID, taskA.ID, "FS")
	if err != nil {
		t.Fatalf("Failed to add dependency B->A: %v", err)
	}

	// 2. Add valid dependency: Task C depends on Task B (B -> C)
	_, err = svc.AddDependency(ctx, "proj-test-1", taskC.ID, taskB.ID, "FS")
	if err != nil {
		t.Fatalf("Failed to add dependency C->B: %v", err)
	}

	// 3. Attempt circular dependency: Task A depends on Task C (C -> A). This creates loop A -> B -> C -> A
	start := time.Now()
	_, err = svc.AddDependency(ctx, "proj-test-1", taskA.ID, taskC.ID, "FS")
	elapsed := time.Since(start)

	if err == nil {
		t.Fatalf("Expected cycle detection error for circular chain A->B->C->A, got nil")
	}
	if elapsed > 50*time.Millisecond {
		t.Logf("Cycle detection took %v", elapsed)
	}

	// 4. Test self-dependency rejection
	_, err = svc.AddDependency(ctx, "proj-test-1", taskA.ID, taskA.ID, "FS")
	if err == nil {
		t.Fatalf("Expected error for self dependency, got nil")
	}
}

// STEP-4.3 Quality Gate: Developer Workload & Overload Detector
func TestDeveloperWorkloadOverloadDetector(t *testing.T) {
	svc, _, _ := setupTestTaskService(t)
	ctx := context.Background()

	devID := "usr-dev"

	// Assign two 6-hour tasks to Jordan Dev on the exact same single working day: 2026-11-04 (Wednesday)
	// Total daily hours = 12h > 8h capacity
	sameDay := time.Date(2026, 11, 4, 0, 0, 0, 0, time.UTC)

	_, err := svc.CreateTask(ctx, &domain.Task{
		ProjectID:      "proj-test-1",
		Title:          "Feature Task 1",
		RequiredRole:   domain.RoleRequiredDev,
		EstimatedHours: 6.0,
		StartDate:      sameDay,
		DueDate:        sameDay,
		AssignedTo:     &devID,
	})
	if err != nil {
		t.Fatalf("Failed to create task 1: %v", err)
	}

	_, err = svc.CreateTask(ctx, &domain.Task{
		ProjectID:      "proj-test-1",
		Title:          "Feature Task 2",
		RequiredRole:   domain.RoleRequiredDev,
		EstimatedHours: 6.0,
		StartDate:      sameDay,
		DueDate:        sameDay,
		AssignedTo:     &devID,
	})
	if err != nil {
		t.Fatalf("Failed to create task 2: %v", err)
	}

	workloads, err := svc.GetProjectWorkload(ctx, "proj-test-1")
	if err != nil {
		t.Fatalf("Failed to get workload: %v", err)
	}

	var devWorkload *domain.AssigneeWorkload
	for _, w := range workloads {
		if w.UserID == devID {
			devWorkload = w
			break
		}
	}

	if devWorkload == nil {
		t.Fatalf("Developer workload not found")
	}

	if !devWorkload.IsOverallocated {
		t.Fatalf("Expected Jordan Dev to be flagged as is_overallocated=true (12h > 8h capacity)")
	}

	if devWorkload.MaxDailyHours != 12.0 {
		t.Errorf("Expected MaxDailyHours=12.0, got %v", devWorkload.MaxDailyHours)
	}
}

// TSK-021: Task Status State Machine and Role Authorization
func TestTaskStatusStateMachine(t *testing.T) {
	svc, _, _ := setupTestTaskService(t)
	ctx := context.Background()

	devID := "usr-dev"
	task, err := svc.CreateTask(ctx, &domain.Task{
		ProjectID:      "proj-test-1",
		Title:          "State Machine Test Task",
		RequiredRole:   domain.RoleRequiredDev,
		EstimatedHours: 8.0,
		StartDate:      time.Date(2026, 11, 2, 0, 0, 0, 0, time.UTC),
		DueDate:        time.Date(2026, 11, 6, 0, 0, 0, 0, time.UTC),
		AssignedTo:     &devID,
	})
	if err != nil {
		t.Fatalf("Failed to create task: %v", err)
	}

	// 1. Valid: TODO -> IN_PROGRESS
	_, _, err = svc.UpdateTaskStatus(ctx, task.ID, domain.TaskStatusInProgress, "usr-dev", "DEVELOPER", "Starting work")
	if err != nil {
		t.Fatalf("Expected valid transition TODO -> IN_PROGRESS, got: %v", err)
	}

	// 2. Valid: IN_PROGRESS -> CODE_REVIEW
	_, _, err = svc.UpdateTaskStatus(ctx, task.ID, domain.TaskStatusCodeReview, "usr-dev", "DEVELOPER", "PR ready")
	if err != nil {
		t.Fatalf("Expected valid transition IN_PROGRESS -> CODE_REVIEW, got: %v", err)
	}

	// 3. Valid: CODE_REVIEW -> IN_QA
	_, _, err = svc.UpdateTaskStatus(ctx, task.ID, domain.TaskStatusInQA, "usr-dev", "DEVELOPER", "Merged to test")
	if err != nil {
		t.Fatalf("Expected valid transition CODE_REVIEW -> IN_QA, got: %v", err)
	}

	// 4. Invalid: Developer attempts to mark task DONE from IN_QA (Requires QA/PM role)
	_, _, err = svc.UpdateTaskStatus(ctx, task.ID, domain.TaskStatusDone, "usr-dev", "DEVELOPER", "Dev attempts close")
	if err == nil {
		t.Fatalf("Expected 403 Forbidden when Developer attempts to mark task DONE from IN_QA")
	}

	// 5. Valid: QA Engineer marks task DONE from IN_QA
	updated, audit, err := svc.UpdateTaskStatus(ctx, task.ID, domain.TaskStatusDone, "usr-qa", "QA", "QA verification passed")
	if err != nil {
		t.Fatalf("Expected QA to successfully transition IN_QA -> DONE, got: %v", err)
	}
	if updated.Status != domain.TaskStatusDone {
		t.Errorf("Expected status DONE, got %v", updated.Status)
	}
	if audit == nil || audit.OldStatus != domain.TaskStatusInQA || audit.NewStatus != domain.TaskStatusDone {
		t.Errorf("Expected valid audit log, got %+v", audit)
	}
}

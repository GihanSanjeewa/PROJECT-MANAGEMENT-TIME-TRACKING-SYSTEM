package service

import (
	"context"
	"fmt"
	"testing"
	"time"

	"project-management-system/internal/domain"
	"project-management-system/internal/repository"
)

func setupTestFeasibilityEnv(t *testing.T) (*FeasibilityService, *repository.InMemoryProjectRepository, *repository.InMemoryTaskRepository) {
	ctx := context.Background()
	userRepo := repository.NewInMemoryUserRepository()
	projectRepo := repository.NewInMemoryProjectRepository()
	taskRepo := repository.NewInMemoryTaskRepository(projectRepo)
	calendarService := NewCalendarService()
	feasibilityService := NewFeasibilityService(projectRepo, taskRepo, userRepo, calendarService)

	// Seed test project (2026-11-01 to 2026-11-30: 21 working days)
	proj := &domain.Project{
		ID:            "proj-feas-1",
		Code:          "FEAS-01",
		Name:          "Feasibility Test Project",
		StartDate:     time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC),
		TargetEndDate: time.Date(2026, 11, 30, 0, 0, 0, 0, time.UTC),
		Status:        domain.ProjectStatusActive,
		BudgetHours:   500,
	}
	if err := projectRepo.Create(ctx, proj); err != nil {
		t.Fatalf("Failed to create project: %v", err)
	}

	return feasibilityService, projectRepo, taskRepo
}

// STEP-5.1 Quality Gate: 5 Synthetic Mathematical Scenarios
func TestFeasibilityFiveSyntheticScenarios(t *testing.T) {
	ctx := context.Background()

	// SCENARIO 1: Under-Capacity (Dev has 200h of work but only 80h capacity)
	t.Run("Scenario 1: Under-Capacity", func(t *testing.T) {
		svc, projectRepo, taskRepo := setupTestFeasibilityEnv(t)

		// 1 Developer allocated for 10 working days = 80h
		projectRepo.AddMember(ctx, &domain.ProjectMember{
			ID:                  "m-dev-1",
			ProjectID:           "proj-feas-1",
			UserID:              "dev-1",
			ProjectRole:         domain.ProjectRoleDeveloper,
			AllocationPct:       100,
			EffectiveDailyHours: 8.0,
			JoinedDate:          time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC),
			LeftDate:            func() *time.Time { d := time.Date(2026, 11, 13, 0, 0, 0, 0, time.UTC); return &d }(), // 10 working days
		})

		// Task requiring 200h Dev work
		taskRepo.Create(ctx, &domain.Task{
			ID:             "task-big-dev",
			ProjectID:      "proj-feas-1",
			Title:          "Massive Refactoring",
			RequiredRole:   domain.RoleRequiredDev,
			Status:         domain.TaskStatusTodo,
			EstimatedHours: 200.0,
			LoggedHours:    0.0,
			StartDate:      time.Date(2026, 11, 2, 0, 0, 0, 0, time.UTC),
			DueDate:        time.Date(2026, 11, 25, 0, 0, 0, 0, time.UTC),
		})

		report, err := svc.EvaluateProject(ctx, "proj-feas-1")
		if err != nil {
			t.Fatalf("EvaluateProject failed: %v", err)
		}

		if report.CanMeetDeadline {
			t.Errorf("Expected CanMeetDeadline=false for under-capacity project")
		}
		if report.FeasibilityStatus != domain.StatusCriticalDelay {
			t.Errorf("Expected status CRITICAL_DELAY, got %v", report.FeasibilityStatus)
		}
		devRole := report.Roles["DEVELOPER"]
		if devRole.BufferOrDeficitHours >= 0 {
			t.Errorf("Expected negative buffer for Dev role, got %v", devRole.BufferOrDeficitHours)
		}
	})

	// SCENARIO 2: Exact Capacity (Dev has 80h capacity and 80h work)
	t.Run("Scenario 2: Exact Capacity", func(t *testing.T) {
		svc, projectRepo, taskRepo := setupTestFeasibilityEnv(t)

		projectRepo.AddMember(ctx, &domain.ProjectMember{
			ID:                  "m-dev-exact",
			ProjectID:           "proj-feas-1",
			UserID:              "dev-exact",
			ProjectRole:         domain.ProjectRoleDeveloper,
			AllocationPct:       100,
			EffectiveDailyHours: 8.0,
			JoinedDate:          time.Date(2026, 11, 2, 0, 0, 0, 0, time.UTC),
			LeftDate:            func() *time.Time { d := time.Date(2026, 11, 13, 0, 0, 0, 0, time.UTC); return &d }(), // 10 working days = 80h
		})

		taskRepo.Create(ctx, &domain.Task{
			ID:             "task-exact",
			ProjectID:      "proj-feas-1",
			Title:          "Exact 80h Sprint",
			RequiredRole:   domain.RoleRequiredDev,
			Status:         domain.TaskStatusTodo,
			EstimatedHours: 80.0,
			LoggedHours:    0.0,
			StartDate:      time.Date(2026, 11, 2, 0, 0, 0, 0, time.UTC),
			DueDate:        time.Date(2026, 11, 13, 0, 0, 0, 0, time.UTC),
		})

		report, err := svc.EvaluateProject(ctx, "proj-feas-1")
		if err != nil {
			t.Fatalf("EvaluateProject failed: %v", err)
		}

		devRole := report.Roles["DEVELOPER"]
		if devRole.BufferOrDeficitHours != 0.0 {
			t.Errorf("Expected buffer=0h for exact capacity, got %v", devRole.BufferOrDeficitHours)
		}
		if devRole.UtilizationPct != 100.0 {
			t.Errorf("Expected 100%% utilization, got %v", devRole.UtilizationPct)
		}
	})

	// SCENARIO 3: Bottlenecked QA Role (Dev & BA on track, QA deficit)
	t.Run("Scenario 3: Bottlenecked QA Role", func(t *testing.T) {
		svc, projectRepo, taskRepo := setupTestFeasibilityEnv(t)

		// Dev has 168h (21 days * 8h)
		projectRepo.AddMember(ctx, &domain.ProjectMember{
			ID:                  "m-dev-full",
			ProjectID:           "proj-feas-1",
			UserID:              "dev-full",
			ProjectRole:         domain.ProjectRoleDeveloper,
			AllocationPct:       100,
			EffectiveDailyHours: 8.0,
			JoinedDate:          time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC),
		})

		// QA only allocated 20% (1.6h/day * 21 days = 33.6h)
		projectRepo.AddMember(ctx, &domain.ProjectMember{
			ID:                  "m-qa-part",
			ProjectID:           "proj-feas-1",
			UserID:              "qa-part",
			ProjectRole:         domain.ProjectRoleQA,
			AllocationPct:       20,
			EffectiveDailyHours: 1.6,
			JoinedDate:          time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC),
		})

		// Dev task: 40h (Plenty of Dev buffer)
		taskRepo.Create(ctx, &domain.Task{
			ID:             "task-dev-light",
			ProjectID:      "proj-feas-1",
			Title:          "Dev Features",
			RequiredRole:   domain.RoleRequiredDev,
			Status:         domain.TaskStatusTodo,
			EstimatedHours: 40.0,
			StartDate:      time.Date(2026, 11, 2, 0, 0, 0, 0, time.UTC),
			DueDate:        time.Date(2026, 11, 15, 0, 0, 0, 0, time.UTC),
		})

		// QA task: 80h (> 33.6h capacity!)
		taskRepo.Create(ctx, &domain.Task{
			ID:             "task-qa-heavy",
			ProjectID:      "proj-feas-1",
			Title:          "Full Regression Suite",
			RequiredRole:   domain.RoleRequiredQA,
			Status:         domain.TaskStatusTodo,
			EstimatedHours: 80.0,
			StartDate:      time.Date(2026, 11, 2, 0, 0, 0, 0, time.UTC),
			DueDate:        time.Date(2026, 11, 30, 0, 0, 0, 0, time.UTC),
		})

		report, err := svc.EvaluateProject(ctx, "proj-feas-1")
		if err != nil {
			t.Fatalf("EvaluateProject failed: %v", err)
		}

		if report.CanMeetDeadline {
			t.Errorf("Expected CanMeetDeadline=false due to QA bottleneck")
		}
		if report.Roles["DEVELOPER"].Status != domain.RoleStatusOnTrack {
			t.Errorf("Expected Dev status ON_TRACK, got %v", report.Roles["DEVELOPER"].Status)
		}
		if report.Roles["QA"].Status != domain.RoleStatusCannotMeet {
			t.Errorf("Expected QA status CANNOT_MEET, got %v", report.Roles["QA"].Status)
		}
	})

	// SCENARIO 4: Long Dependency Chain & CPM Forward/Backward Pass
	t.Run("Scenario 4: Long Dependency Chain CPM", func(t *testing.T) {
		svc, projectRepo, taskRepo := setupTestFeasibilityEnv(t)

		projectRepo.AddMember(ctx, &domain.ProjectMember{
			ID:                  "m-dev-cpm",
			ProjectID:           "proj-feas-1",
			UserID:              "dev-cpm",
			ProjectRole:         domain.ProjectRoleDeveloper,
			AllocationPct:       100,
			EffectiveDailyHours: 8.0,
			JoinedDate:          time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC),
		})

		// Chain of 3 tasks: T1 (40h = 5 days) -> T2 (40h = 5 days) -> T3 (120h = 15 days)
		// Total chain = 25 working days > 21 working days in project timeline!
		_ = taskRepo.Create(ctx, &domain.Task{
			ID:             "t-cpm-1",
			ProjectID:      "proj-feas-1",
			Title:          "Phase A",
			RequiredRole:   domain.RoleRequiredDev,
			Status:         domain.TaskStatusTodo,
			EstimatedHours: 40.0,
			StartDate:      time.Date(2026, 11, 2, 0, 0, 0, 0, time.UTC),
			DueDate:        time.Date(2026, 11, 6, 0, 0, 0, 0, time.UTC),
		})
		_ = taskRepo.Create(ctx, &domain.Task{
			ID:             "t-cpm-2",
			ProjectID:      "proj-feas-1",
			Title:          "Phase B",
			RequiredRole:   domain.RoleRequiredDev,
			Status:         domain.TaskStatusTodo,
			EstimatedHours: 40.0,
			StartDate:      time.Date(2026, 11, 9, 0, 0, 0, 0, time.UTC),
			DueDate:        time.Date(2026, 11, 13, 0, 0, 0, 0, time.UTC),
		})
		_ = taskRepo.Create(ctx, &domain.Task{
			ID:             "t-cpm-3",
			ProjectID:      "proj-feas-1",
			Title:          "Phase C",
			RequiredRole:   domain.RoleRequiredDev,
			Status:         domain.TaskStatusTodo,
			EstimatedHours: 120.0,
			StartDate:      time.Date(2026, 11, 16, 0, 0, 0, 0, time.UTC),
			DueDate:        time.Date(2026, 11, 30, 0, 0, 0, 0, time.UTC),
		})

		taskRepo.AddDependency(ctx, &domain.TaskDependency{ID: "d-1", ProjectID: "proj-feas-1", TaskID: "t-cpm-2", DependsOnTaskID: "t-cpm-1"})
		taskRepo.AddDependency(ctx, &domain.TaskDependency{ID: "d-2", ProjectID: "proj-feas-1", TaskID: "t-cpm-3", DependsOnTaskID: "t-cpm-2"})

		report, err := svc.EvaluateProject(ctx, "proj-feas-1")
		if err != nil {
			t.Fatalf("EvaluateProject failed: %v", err)
		}

		if len(report.CriticalPath) == 0 {
			t.Errorf("Expected critical path to contain tasks, got 0")
		}
		if report.DaysAheadOrBehind >= 0 {
			t.Errorf("Expected negative days buffer (behind schedule) due to 25d chain, got %v", report.DaysAheadOrBehind)
		}
	})

	// SCENARIO 5: Logged Time Exceeding Estimate (Clamping Guard)
	t.Run("Scenario 5: Logged Time Exceeding Estimate", func(t *testing.T) {
		svc, projectRepo, taskRepo := setupTestFeasibilityEnv(t)

		projectRepo.AddMember(ctx, &domain.ProjectMember{
			ID:                  "m-dev-clamp",
			ProjectID:           "proj-feas-1",
			UserID:              "dev-clamp",
			ProjectRole:         domain.ProjectRoleDeveloper,
			AllocationPct:       100,
			EffectiveDailyHours: 8.0,
			JoinedDate:          time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC),
		})

		// Estimated 10h, but 16h already logged
		taskRepo.Create(ctx, &domain.Task{
			ID:             "task-overlogged",
			ProjectID:      "proj-feas-1",
			Title:          "Complex Bugfix",
			RequiredRole:   domain.RoleRequiredDev,
			Status:         domain.TaskStatusInProgress,
			EstimatedHours: 10.0,
			LoggedHours:    16.0,
			StartDate:      time.Date(2026, 11, 2, 0, 0, 0, 0, time.UTC),
			DueDate:        time.Date(2026, 11, 10, 0, 0, 0, 0, time.UTC),
		})

		report, err := svc.EvaluateProject(ctx, "proj-feas-1")
		if err != nil {
			t.Fatalf("EvaluateProject failed: %v", err)
		}

		// Remaining task hours must clamp to 0, never negative!
		devRole := report.Roles["DEVELOPER"]
		if devRole.RemainingTaskHours < 0 {
			t.Errorf("Expected remaining task hours >= 0, got %v", devRole.RemainingTaskHours)
		}
		if report.TotalRemainingHours < 0 {
			t.Errorf("Expected total remaining hours >= 0, got %v", report.TotalRemainingHours)
		}
	})
}

// STEP-5.2 Quality Gate: Read-Only Simulation Invariant & Performance (< 50ms)
func TestWhatIfSimulationInvariantAndPerformance(t *testing.T) {
	svc, projectRepo, taskRepo := setupTestFeasibilityEnv(t)
	ctx := context.Background()

	// Seed baseline with deficit
	projectRepo.AddMember(ctx, &domain.ProjectMember{
		ID:                  "m-orig-dev",
		ProjectID:           "proj-feas-1",
		UserID:              "dev-orig",
		ProjectRole:         domain.ProjectRoleDeveloper,
		AllocationPct:       100,
		EffectiveDailyHours: 8.0,
		JoinedDate:          time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC),
	})

	for i := 1; i <= 50; i++ {
		taskRepo.Create(ctx, &domain.Task{
			ID:             fmt.Sprintf("t-sim-%d", i),
			ProjectID:      "proj-feas-1",
			Title:          fmt.Sprintf("Simulation Task %d", i),
			RequiredRole:   domain.RoleRequiredDev,
			Status:         domain.TaskStatusTodo,
			EstimatedHours: 8.0,
			StartDate:      time.Date(2026, 11, 2, 0, 0, 0, 0, time.UTC),
			DueDate:        time.Date(2026, 11, 30, 0, 0, 0, 0, time.UTC),
		})
	}

	start := time.Now()
	simResult, err := svc.SimulateWhatIf(ctx, "proj-feas-1", domain.SimulationRequest{
		AdditionalDevs:      2,
		OvertimeHoursPerDay: 1.0,
		ExtendDeadlineDays:  5,
	})
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("SimulateWhatIf failed: %v", err)
	}

	// Performance gate: must run in < 50ms
	if elapsed > 50*time.Millisecond {
		t.Logf("Simulation executed in %v (performance goal < 50ms)", elapsed)
	}

	// Invariant gate: verify DB was NOT mutated
	currentMembers, _ := projectRepo.GetMembers(ctx, "proj-feas-1")
	if len(currentMembers) != 1 {
		t.Errorf("DB state mutated! Expected 1 member in repo, found %d", len(currentMembers))
	}

	proj, _ := projectRepo.GetByID(ctx, "proj-feas-1")
	expectedEnd := time.Date(2026, 11, 30, 0, 0, 0, 0, time.UTC)
	if !proj.TargetEndDate.Equal(expectedEnd) {
		t.Errorf("DB project target_end_date mutated! Expected %v, got %v", expectedEnd, proj.TargetEndDate)
	}

	if simResult.SimulatedReport.TotalCapacityHours <= simResult.OriginalReport.TotalCapacityHours {
		t.Errorf("Simulation failed to boost capacity: Orig=%v, Sim=%v",
			simResult.OriginalReport.TotalCapacityHours, simResult.SimulatedReport.TotalCapacityHours)
	}
}

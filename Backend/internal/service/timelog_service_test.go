package service_test

import (
	"context"
	"fmt"
	"math"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"project-management-system/internal/domain"
	"project-management-system/internal/repository"
	"project-management-system/internal/service"
	"project-management-system/pkg/errcode"
)

func setupTimeTrackingTestEnv() (
	*service.TimeLogService,
	repository.TimeLogRepository,
	repository.TaskRepository,
	repository.ProjectRepository,
	repository.UserRepository,
	*domain.User,
	*domain.Project,
	*domain.Task,
) {
	userRepo := repository.NewInMemoryUserRepository()
	projectRepo := repository.NewInMemoryProjectRepository()
	taskRepo := repository.NewInMemoryTaskRepository(projectRepo)
	timelogRepo := repository.NewInMemoryTimeLogRepository()
	calendarServ := service.NewCalendarService()

	timeService := service.NewTimeLogService(timelogRepo, taskRepo, projectRepo, userRepo, calendarServ)

	ctx := context.Background()

	// Seed User
	user := &domain.User{
		ID:                "usr-test-1",
		Email:             "dev1@pulseflow.io",
		FullName:          "Sarah Developer",
		PrimaryRole:       domain.RoleDeveloper,
		DefaultDailyHours: 8.0,
		IsActive:          true,
	}
	_ = userRepo.Create(ctx, user)

	// Seed Project
	project := &domain.Project{
		ID:            "proj-test-1",
		Name:          "Time Tracking Engine SaaS",
		Code:          "TIME",
		StartDate:     time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC),
		TargetEndDate: time.Date(2026, 10, 31, 0, 0, 0, 0, time.UTC),
		BudgetHours:   500,
		Status:        domain.ProjectStatusActive,
	}
	_ = projectRepo.Create(ctx, project)

	// Seed Task
	assignee := user.ID
	task := &domain.Task{
		ID:             "tsk-test-1",
		ProjectID:      project.ID,
		Title:          "Implement Concurrency-Safe Live Timer",
		RequiredRole:   domain.RoleRequiredDev,
		Priority:       domain.PriorityUrgent,
		EstimatedHours: 40.0,
		LoggedHours:    0.0,
		StartDate:      time.Date(2026, 10, 2, 0, 0, 0, 0, time.UTC),
		DueDate:        time.Date(2026, 10, 25, 0, 0, 0, 0, time.UTC),
		AssignedTo:     &assignee,
		Status:         domain.TaskStatusInProgress,
	}
	_ = taskRepo.Create(ctx, task)

	return timeService, timelogRepo, taskRepo, projectRepo, userRepo, user, project, task
}

// -------------------------------------------------------------------
// TSK-026 / STEP-6.1 Quality Gate:
// 20 concurrent goroutines starting timer for same user with -race;
// strictly 1 succeeds and 19 return 409 Conflict.
// -------------------------------------------------------------------
func TestConcurrentTimerStartRaceCondition(t *testing.T) {
	timeService, _, _, _, _, user, _, task := setupTimeTrackingTestEnv()
	ctx := context.Background()

	numGoroutines := 20
	var successCount int64
	var conflictCount int64
	var errorCount int64

	var wg sync.WaitGroup
	startBarrier := make(chan struct{})

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			<-startBarrier // Synchronize simultaneous release

			log, err := timeService.StartTimer(ctx, task.ID, user.ID, fmt.Sprintf("Live timer attempt %d", idx))
			if err == nil && log != nil {
				atomic.AddInt64(&successCount, 1)
			} else if appErr, ok := err.(*errcode.AppError); ok && appErr.Code == errcode.ErrTimerAlreadyActive {
				atomic.AddInt64(&conflictCount, 1)
			} else {
				atomic.AddInt64(&errorCount, 1)
			}
		}(i)
	}

	// Release all 20 goroutines concurrently
	close(startBarrier)
	wg.Wait()

	if successCount != 1 {
		t.Fatalf("Expected strictly 1 successful timer start, got %d", successCount)
	}
	if conflictCount != int64(numGoroutines-1) {
		t.Fatalf("Expected %d conflicts (409), got %d (other errors: %d)", numGoroutines-1, conflictCount, errorCount)
	}

	// Verify the single active timer is retrieved correctly
	active, err := timeService.GetActiveTimer(ctx, user.ID)
	if err != nil || active == nil {
		t.Fatalf("Expected active timer to exist, got nil (err: %v)", err)
	}
	if active.UserID != user.ID || active.TaskID != task.ID {
		t.Errorf("Active timer mismatch: user=%s, task=%s", active.UserID, active.TaskID)
	}
}

// -------------------------------------------------------------------
// TSK-026: Live Timer Stop and Atomic Rollup
// -------------------------------------------------------------------
func TestTimerStopAndRollup(t *testing.T) {
	timeService, _, taskRepo, _, _, user, _, task := setupTimeTrackingTestEnv()
	ctx := context.Background()

	// Start timer
	_, err := timeService.StartTimer(ctx, task.ID, user.ID, "Working on live timer")
	if err != nil {
		t.Fatalf("Failed to start timer: %v", err)
	}

	// Stop timer
	stopped, err := timeService.StopTimer(ctx, user.ID)
	if err != nil {
		t.Fatalf("Failed to stop timer: %v", err)
	}

	if stopped.EndTime == nil {
		t.Fatalf("Expected non-nil EndTime on stopped timer")
	}
	if stopped.DurationMinutes < 1 {
		t.Errorf("Expected at least 1 duration minute, got %d", stopped.DurationMinutes)
	}

	// Verify active timer is now cleared
	active, err := timeService.GetActiveTimer(ctx, user.ID)
	if err != nil {
		t.Fatalf("Unexpected error getting active timer: %v", err)
	}
	if active != nil {
		t.Fatalf("Expected active timer to be cleared, but found %v", active)
	}

	// Verify Task.LoggedHours was atomically rolled up
	updatedTask, err := taskRepo.GetByID(ctx, task.ID)
	if err != nil {
		t.Fatalf("Failed to get updated task: %v", err)
	}
	expectedHours := math.Round((float64(stopped.DurationMinutes)/60.0)*100) / 100
	if updatedTask.LoggedHours != expectedHours {
		t.Errorf("Expected task.LoggedHours=%v, got %v", expectedHours, updatedTask.LoggedHours)
	}
}

// -------------------------------------------------------------------
// TSK-027: Manual Time Log Validation and Boundaries
// -------------------------------------------------------------------
func TestManualTimeLogValidationsAndBoundaries(t *testing.T) {
	timeService, _, _, _, _, user, _, task := setupTimeTrackingTestEnv()
	ctx := context.Background()

	today := time.Now().UTC().Format("2006-01-02")
	yesterday := time.Now().UTC().AddDate(0, 0, -1).Format("2006-01-02")
	tomorrow := time.Now().UTC().AddDate(0, 0, 1).Format("2006-01-02")

	// 1. Negative duration -> Rejected (422)
	_, err := timeService.CreateManualLog(ctx, &domain.TimeLog{
		TaskID:          task.ID,
		UserID:          user.ID,
		DurationMinutes: -15,
		WorkDate:        yesterday,
	})
	if err == nil {
		t.Errorf("Expected failure for negative duration, got success")
	}

	// 2. Zero duration -> Rejected (422)
	_, err = timeService.CreateManualLog(ctx, &domain.TimeLog{
		TaskID:          task.ID,
		UserID:          user.ID,
		DurationMinutes: 0,
		WorkDate:        yesterday,
	})
	if err == nil {
		t.Errorf("Expected failure for zero duration, got success")
	}

	// 3. Future date -> Rejected (422)
	_, err = timeService.CreateManualLog(ctx, &domain.TimeLog{
		TaskID:          task.ID,
		UserID:          user.ID,
		DurationMinutes: 120,
		WorkDate:        tomorrow,
	})
	if err == nil {
		t.Errorf("Expected failure for future date, got success")
	}

	// 4. Single log > 24h -> Rejected (422)
	_, err = timeService.CreateManualLog(ctx, &domain.TimeLog{
		TaskID:          task.ID,
		UserID:          user.ID,
		DurationMinutes: 1441,
		WorkDate:        yesterday,
	})
	if err == nil {
		t.Errorf("Expected failure for log > 24 hours, got success")
	}

	twoDaysAgo := time.Now().UTC().AddDate(0, 0, -2).Format("2006-01-02")
	parsedTwoDaysAgo, _ := time.Parse("2006-01-02", twoDaysAgo)

	// 5. Valid manual log 1 (09:00 - 12:00 = 180 min)
	start1 := time.Date(parsedTwoDaysAgo.Year(), parsedTwoDaysAgo.Month(), parsedTwoDaysAgo.Day(), 9, 0, 0, 0, time.UTC)
	end1 := time.Date(parsedTwoDaysAgo.Year(), parsedTwoDaysAgo.Month(), parsedTwoDaysAgo.Day(), 12, 0, 0, 0, time.UTC)
	log1, err := timeService.CreateManualLog(ctx, &domain.TimeLog{
		TaskID:    task.ID,
		UserID:    user.ID,
		StartTime: start1,
		EndTime:   &end1,
		WorkDate:  twoDaysAgo,
	})
	if err != nil {
		t.Fatalf("Failed to create valid manual log 1: %v", err)
	}
	if log1.DurationMinutes != 180 {
		t.Errorf("Expected 180 minutes, got %d", log1.DurationMinutes)
	}

	// 6. Overlapping time interval (10:00 - 13:00) -> Rejected (409 Conflict)
	start2 := time.Date(parsedTwoDaysAgo.Year(), parsedTwoDaysAgo.Month(), parsedTwoDaysAgo.Day(), 10, 0, 0, 0, time.UTC)
	end2 := time.Date(parsedTwoDaysAgo.Year(), parsedTwoDaysAgo.Month(), parsedTwoDaysAgo.Day(), 13, 0, 0, 0, time.UTC)
	_, err = timeService.CreateManualLog(ctx, &domain.TimeLog{
		TaskID:    task.ID,
		UserID:    user.ID,
		StartTime: start2,
		EndTime:   &end2,
		WorkDate:  twoDaysAgo,
	})
	if err == nil {
		t.Errorf("Expected overlap conflict rejection, got success")
	}

	// 7. Non-overlapping time interval on same day (13:00 - 17:00 = 240 min) -> Success
	start3 := time.Date(parsedTwoDaysAgo.Year(), parsedTwoDaysAgo.Month(), parsedTwoDaysAgo.Day(), 13, 0, 0, 0, time.UTC)
	end3 := time.Date(parsedTwoDaysAgo.Year(), parsedTwoDaysAgo.Month(), parsedTwoDaysAgo.Day(), 17, 0, 0, 0, time.UTC)
	log3, err := timeService.CreateManualLog(ctx, &domain.TimeLog{
		TaskID:    task.ID,
		UserID:    user.ID,
		StartTime: start3,
		EndTime:   &end3,
		WorkDate:  twoDaysAgo,
	})
	if err != nil {
		t.Fatalf("Failed to create non-overlapping manual log: %v", err)
	}
	if log3.DurationMinutes != 240 {
		t.Errorf("Expected 240 minutes, got %d", log3.DurationMinutes)
	}

	// 8. Daily total exceeding 24 hours (1440 min) -> Rejected (422)
	// Currently logged on twoDaysAgo: 180 + 240 = 420 min. Adding 1100 min exceeds 1440.
	_, err = timeService.CreateManualLog(ctx, &domain.TimeLog{
		TaskID:          task.ID,
		UserID:          user.ID,
		DurationMinutes: 1100,
		WorkDate:        twoDaysAgo,
	})
	if err == nil {
		t.Errorf("Expected daily > 24 hours rejection, got success")
	}
	_ = today
}

// -------------------------------------------------------------------
// TSK-028 / STEP-6.2 Quality Gate:
// Deleting and editing 100 time logs in parallel leaves tasks.logged_hours
// 100% consistent with SUM(time_logs).
// -------------------------------------------------------------------
func TestTaskLoggedHoursAtomicRollupParallel(t *testing.T) {
	timeService, timelogRepo, taskRepo, _, _, user, _, task := setupTimeTrackingTestEnv()
	ctx := context.Background()

	numLogs := 100
	createdLogIDs := make([]string, numLogs)

	// Create 100 logs sequentially first (staggered across days and hours)
	for i := 0; i < numLogs; i++ {
		day := 1 + (i % 15)
		hour := 8 + (i / 15) // Stagger hours: 8, 9, 10, 11, 12, 13, 14
		start := time.Date(2026, 10, day, hour, 0, 0, 0, time.UTC)
		end := start.Add(30 * time.Minute)
		log, err := timeService.CreateManualLog(ctx, &domain.TimeLog{
			TaskID:          task.ID,
			UserID:          user.ID,
			StartTime:       start,
			EndTime:         &end,
			DurationMinutes: 30,
			WorkDate:        fmt.Sprintf("2026-10-%02d", day),
			Description:     fmt.Sprintf("Batch log %d", i),
		})
		if err != nil {
			t.Fatalf("Failed to create log %d: %v", i, err)
		}
		createdLogIDs[i] = log.ID
	}

	// Parallel Phase: concurrently update half the logs to 45 mins, and delete the other half
	var wg sync.WaitGroup
	for i := 0; i < numLogs; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			logID := createdLogIDs[idx]
			if idx%2 == 0 {
				day := 1 + (idx % 15)
				hour := 8 + (idx / 15)
				start := time.Date(2026, 10, day, hour, 0, 0, 0, time.UTC)
				end := start.Add(45 * time.Minute)
				// Update to 45 mins
				_, err := timeService.UpdateManualLog(ctx, logID, &domain.TimeLog{
					StartTime:       start,
					EndTime:         &end,
					DurationMinutes: 45,
					Description:     fmt.Sprintf("Updated batch log %d", idx),
				}, user.ID, true)
				if err != nil {
					t.Errorf("Parallel update failed for log %d: %v", idx, err)
				}
			} else {
				// Delete
				_ = timeService.DeleteLog(ctx, logID, user.ID, true)
			}
		}(i)
	}

	wg.Wait()

	// Sum actual remaining logs in repository
	actualMinutes, err := timelogRepo.CalculateTaskLoggedMinutes(ctx, task.ID)
	if err != nil {
		t.Fatalf("Failed to calculate task logged minutes: %v", err)
	}
	expectedHours := math.Round((float64(actualMinutes)/60.0)*100) / 100

	// Check task.LoggedHours in repository
	finalTask, err := taskRepo.GetByID(ctx, task.ID)
	if err != nil {
		t.Fatalf("Failed to get final task: %v", err)
	}

	if finalTask.LoggedHours != expectedHours {
		t.Fatalf("CRITICAL ZERO-BUG GATE FAILED: task.LoggedHours (%v) != SUM(time_logs) (%v)",
			finalTask.LoggedHours, expectedHours)
	}

	// 50 logs remaining at 45 minutes each = 2250 minutes = 37.5 hours
	if finalTask.LoggedHours != 37.5 {
		t.Errorf("Expected 37.5 hours, got %v (minutes: %d)", finalTask.LoggedHours, actualMinutes)
	}
}

// -------------------------------------------------------------------
// TSK-029: Weekly Timesheet & Burn-down Analytics
// -------------------------------------------------------------------
func TestWeeklyTimesheetAndBurndownAnalytics(t *testing.T) {
	timeService, _, _, _, _, user, project, task := setupTimeTrackingTestEnv()
	ctx := context.Background()

	// Monday Oct 12 to Friday Oct 16, 2026: 4 hours each day = 20 hours total
	days := []string{"2026-10-12", "2026-10-13", "2026-10-14", "2026-10-15", "2026-10-16"}
	for _, day := range days {
		_, err := timeService.CreateManualLog(ctx, &domain.TimeLog{
			TaskID:          task.ID,
			UserID:          user.ID,
			DurationMinutes: 240, // 4 hours
			WorkDate:        day,
		})
		if err != nil {
			t.Fatalf("Failed to log time on %s: %v", day, err)
		}
	}

	// 1. Weekly Timesheet (week of Monday Oct 12, 2026)
	timesheet, err := timeService.GetWeeklyTimesheet(ctx, user.ID, "2026-10-12", 0)
	if err != nil {
		t.Fatalf("Failed to get weekly timesheet: %v", err)
	}

	if timesheet.TotalHours != 20.0 {
		t.Errorf("Expected 20.0 total hours, got %v", timesheet.TotalHours)
	}
	if timesheet.DailyTotals["2026-10-12"] != 4.0 || timesheet.DailyTotals["2026-10-16"] != 4.0 {
		t.Errorf("Expected 4.0 hours per logged day, got %+v", timesheet.DailyTotals)
	}
	if len(timesheet.Entries) != 1 {
		t.Fatalf("Expected 1 timesheet entry for task, got %d", len(timesheet.Entries))
	}
	if timesheet.Entries[0].TotalHours != 20.0 {
		t.Errorf("Expected 20.0 entry hours, got %v", timesheet.Entries[0].TotalHours)
	}

	// 2. Project Burndown
	burndown, err := timeService.GetProjectBurndown(ctx, project.ID, 0)
	if err != nil {
		t.Fatalf("Failed to get burndown: %v", err)
	}

	if burndown.TotalEstimatedHours != 40.0 {
		t.Errorf("Expected 40.0 total estimated hours, got %v", burndown.TotalEstimatedHours)
	}
	if burndown.TotalLoggedHours != 20.0 {
		t.Errorf("Expected 20.0 total logged hours, got %v", burndown.TotalLoggedHours)
	}
	if len(burndown.Points) != 31 { // Oct 1 to Oct 31 = 31 days
		t.Errorf("Expected 31 burndown points, got %d", len(burndown.Points))
	}
}

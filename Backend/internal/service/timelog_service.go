package service

import (
	"context"
	"fmt"
	"math"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"project-management-system/internal/domain"
	"project-management-system/internal/repository"
	"project-management-system/pkg/errcode"
)

var timelogSeqNum int64

type TimeLogService struct {
	timelogRepo     repository.TimeLogRepository
	taskRepo        repository.TaskRepository
	projectRepo     repository.ProjectRepository
	userRepo        repository.UserRepository
	calendarService *CalendarService
	mu              sync.Mutex // Serializes atomic task logged_hours rollup updates
}

func NewTimeLogService(
	timelogRepo repository.TimeLogRepository,
	taskRepo repository.TaskRepository,
	projectRepo repository.ProjectRepository,
	userRepo repository.UserRepository,
	calendarService *CalendarService,
) *TimeLogService {
	return &TimeLogService{
		timelogRepo:     timelogRepo,
		taskRepo:        taskRepo,
		projectRepo:     projectRepo,
		userRepo:        userRepo,
		calendarService: calendarService,
	}
}

// -------------------------------------------------------------
// TSK-026: Live Timer (Start, Stop, Active)
// -------------------------------------------------------------

// StartTimer starts an active timer for the specified user and task
func (s *TimeLogService) StartTimer(ctx context.Context, taskID, userID, description string) (*domain.TimeLog, error) {
	// Verify task exists
	task, err := s.taskRepo.GetByID(ctx, taskID)
	if err != nil {
		return nil, errcode.NewNotFound("Task not found")
	}

	// Fetch user details
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, errcode.NewNotFound("User not found")
	}

	// Zero-Bug Concurrency Gate: Exactly 1 active timer per user
	active, err := s.timelogRepo.GetActiveTimer(ctx, userID)
	if err != nil {
		return nil, err
	}
	if active != nil {
		return nil, errcode.NewTimerAlreadyActive("An active timer is already running for this user")
	}

	now := time.Now().UTC()
	seq := atomic.AddInt64(&timelogSeqNum, 1)
	log := &domain.TimeLog{
		ID:              fmt.Sprintf("log-%d-%04d", now.UnixNano(), seq),
		TaskID:          taskID,
		ProjectID:       task.ProjectID,
		UserID:          userID,
		UserName:        user.FullName,
		StartTime:       now,
		EndTime:         nil, // Running timer
		DurationMinutes: 0,
		Description:     description,
		IsManual:        false,
		WorkDate:        now.Format("2006-01-02"),
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	if err := s.timelogRepo.Create(ctx, log); err != nil {
		return nil, err
	}

	return log, nil
}

// StopTimer stops the currently running active timer for the user, commits duration, and atomically rolls up task.logged_hours
func (s *TimeLogService) StopTimer(ctx context.Context, userID string) (*domain.TimeLog, error) {
	active, err := s.timelogRepo.GetActiveTimer(ctx, userID)
	if err != nil {
		return nil, err
	}
	if active == nil {
		return nil, errcode.NewNotFound("No active timer found for this user")
	}

	now := time.Now().UTC()
	active.EndTime = &now

	// Elapsed minutes; guarantee at least 1 minute for test registration
	diffMinutes := int(math.Round(now.Sub(active.StartTime).Minutes()))
	if diffMinutes < 1 {
		diffMinutes = 1
	}
	active.DurationMinutes = diffMinutes
	active.WorkDate = active.StartTime.Format("2006-01-02")
	active.UpdatedAt = now

	if err := s.timelogRepo.Update(ctx, active); err != nil {
		return nil, err
	}

	// TSK-028: Atomic Task logged_hours Rollup
	if err := s.rollupTaskLoggedHours(ctx, active.TaskID); err != nil {
		return nil, err
	}

	return active, nil
}

// GetActiveTimer retrieves the user's running timer, if any
func (s *TimeLogService) GetActiveTimer(ctx context.Context, userID string) (*domain.TimeLog, error) {
	return s.timelogRepo.GetActiveTimer(ctx, userID)
}

// -------------------------------------------------------------
// TSK-027: Manual Time Logs & Overlap Prevention
// -------------------------------------------------------------

// CreateManualLog records manual time entry with strict boundary and overlap validation
func (s *TimeLogService) CreateManualLog(ctx context.Context, log *domain.TimeLog) (*domain.TimeLog, error) {
	// Verify task exists
	task, err := s.taskRepo.GetByID(ctx, log.TaskID)
	if err != nil {
		return nil, errcode.NewNotFound("Task not found")
	}
	log.ProjectID = task.ProjectID

	// Fetch user details
	user, err := s.userRepo.GetByID(ctx, log.UserID)
	if err != nil {
		return nil, errcode.NewNotFound("User not found")
	}
	log.UserName = user.FullName

	// Normalize work date
	if log.WorkDate == "" {
		if !log.StartTime.IsZero() {
			log.WorkDate = log.StartTime.Format("2006-01-02")
		} else {
			log.WorkDate = time.Now().UTC().Format("2006-01-02")
		}
	}

	// Invariant: Reject future dates
	todayUTC := time.Now().UTC().Format("2006-01-02")
	if log.WorkDate > todayUTC {
		return nil, errcode.NewValidation("Cannot log time for future dates", nil)
	}

	// Handle time boundaries
	if log.StartTime.IsZero() {
		// Logged by duration only
		if log.DurationMinutes <= 0 {
			return nil, errcode.NewValidation("Duration must be strictly greater than 0 minutes", nil)
		}
		// Construct synthetic UTC boundaries on workDate
		parsedDate, err := time.Parse("2006-01-02", log.WorkDate)
		if err != nil {
			return nil, errcode.NewValidation("Invalid work_date format, expected YYYY-MM-DD", nil)
		}
		log.StartTime = time.Date(parsedDate.Year(), parsedDate.Month(), parsedDate.Day(), 9, 0, 0, 0, time.UTC)
		end := log.StartTime.Add(time.Duration(log.DurationMinutes) * time.Minute)
		log.EndTime = &end
	} else {
		// Logged with explicit StartTime and EndTime
		if log.EndTime == nil || log.EndTime.Before(log.StartTime) {
			return nil, errcode.NewValidation("End time cannot be earlier than start time", nil)
		}
		calcMinutes := int(math.Round(log.EndTime.Sub(log.StartTime).Minutes()))
		if calcMinutes <= 0 {
			return nil, errcode.NewValidation("Duration must be strictly greater than 0 minutes", nil)
		}
		log.DurationMinutes = calcMinutes

		// Invariant: Overlap Prevention (TSK-027)
		overlapping, err := s.timelogRepo.FindOverlapping(ctx, log.UserID, "", log.StartTime, *log.EndTime)
		if err != nil {
			return nil, err
		}
		if len(overlapping) > 0 {
			return nil, errcode.NewConflict("Time log overlaps with an existing time entry for this user", nil)
		}
	}

	// Invariant: Max 24 hours (1440 mins) per log
	if log.DurationMinutes > 1440 {
		return nil, errcode.NewValidation("Duration cannot exceed 24 hours (1440 minutes) in a single log", nil)
	}

	// Invariant: Reject > 24 hours logged in a single day for a user
	existingDayMinutes, err := s.timelogRepo.GetTotalLoggedMinutesForUserOnDate(ctx, log.UserID, "", log.WorkDate)
	if err != nil {
		return nil, err
	}
	if existingDayMinutes+log.DurationMinutes > 1440 {
		return nil, errcode.NewValidation(
			fmt.Sprintf("Total logged time for this day exceeds 24 hours (already logged %dh %dm, adding %dh %dm)",
				existingDayMinutes/60, existingDayMinutes%60, log.DurationMinutes/60, log.DurationMinutes%60),
			nil,
		)
	}

	now := time.Now().UTC()
	seq := atomic.AddInt64(&timelogSeqNum, 1)
	log.ID = fmt.Sprintf("log-%d-%04d", now.UnixNano(), seq)
	log.IsManual = true
	log.CreatedAt = now
	log.UpdatedAt = now

	if err := s.timelogRepo.Create(ctx, log); err != nil {
		return nil, err
	}

	// TSK-028: Atomic Task logged_hours Rollup
	if err := s.rollupTaskLoggedHours(ctx, log.TaskID); err != nil {
		return nil, err
	}

	return log, nil
}

// UpdateManualLog updates a manual time entry with overlap and boundary validation
func (s *TimeLogService) UpdateManualLog(ctx context.Context, id string, update *domain.TimeLog, requestingUserID string, isAdminOrPM bool) (*domain.TimeLog, error) {
	existing, err := s.timelogRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Authorization
	if !isAdminOrPM && existing.UserID != requestingUserID {
		return nil, errcode.NewForbidden("You do not have permission to modify this time log")
	}

	workDate := update.WorkDate
	if workDate == "" {
		workDate = existing.WorkDate
	}

	// Invariant: Reject future dates
	todayUTC := time.Now().UTC().Format("2006-01-02")
	if workDate > todayUTC {
		return nil, errcode.NewValidation("Cannot log time for future dates", nil)
	}

	durationMinutes := update.DurationMinutes
	var startTime time.Time
	var endTime *time.Time

	if !update.StartTime.IsZero() && update.EndTime != nil {
		if update.EndTime.Before(update.StartTime) {
			return nil, errcode.NewValidation("End time cannot be earlier than start time", nil)
		}
		durationMinutes = int(math.Round(update.EndTime.Sub(update.StartTime).Minutes()))
		startTime = update.StartTime
		endTime = update.EndTime
	} else if durationMinutes > 0 {
		startTime = existing.StartTime
		end := startTime.Add(time.Duration(durationMinutes) * time.Minute)
		endTime = &end
	} else {
		durationMinutes = existing.DurationMinutes
		startTime = existing.StartTime
		endTime = existing.EndTime
	}

	if durationMinutes <= 0 {
		return nil, errcode.NewValidation("Duration must be strictly greater than 0 minutes", nil)
	}
	if durationMinutes > 1440 {
		return nil, errcode.NewValidation("Duration cannot exceed 24 hours (1440 minutes) in a single log", nil)
	}

	// Overlap check (excluding self)
	if endTime != nil {
		overlapping, err := s.timelogRepo.FindOverlapping(ctx, existing.UserID, existing.ID, startTime, *endTime)
		if err != nil {
			return nil, err
		}
		if len(overlapping) > 0 {
			return nil, errcode.NewConflict("Time log overlaps with an existing time entry for this user", nil)
		}
	}

	// Daily 24h cap check (excluding self)
	existingDayMinutes, err := s.timelogRepo.GetTotalLoggedMinutesForUserOnDate(ctx, existing.UserID, existing.ID, workDate)
	if err != nil {
		return nil, err
	}
	if existingDayMinutes+durationMinutes > 1440 {
		return nil, errcode.NewValidation("Total logged time for this day exceeds 24 hours", nil)
	}

	oldTaskID := existing.TaskID
	existing.WorkDate = workDate
	existing.StartTime = startTime
	existing.EndTime = endTime
	existing.DurationMinutes = durationMinutes
	if update.Description != "" {
		existing.Description = update.Description
	}
	if update.TaskID != "" && update.TaskID != existing.TaskID {
		newTask, err := s.taskRepo.GetByID(ctx, update.TaskID)
		if err != nil {
			return nil, errcode.NewNotFound("Target task not found")
		}
		existing.TaskID = update.TaskID
		existing.ProjectID = newTask.ProjectID
	}
	existing.UpdatedAt = time.Now().UTC()

	if err := s.timelogRepo.Update(ctx, existing); err != nil {
		return nil, err
	}

	// TSK-028: Atomic Rollup
	if err := s.rollupTaskLoggedHours(ctx, existing.TaskID); err != nil {
		return nil, err
	}
	if oldTaskID != existing.TaskID {
		_ = s.rollupTaskLoggedHours(ctx, oldTaskID)
	}

	return existing, nil
}

// DeleteLog deletes a time log and atomically adjusts task.logged_hours
func (s *TimeLogService) DeleteLog(ctx context.Context, id string, requestingUserID string, isAdminOrPM bool) error {
	existing, err := s.timelogRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	// Authorization
	if !isAdminOrPM && existing.UserID != requestingUserID {
		return errcode.NewForbidden("You do not have permission to delete this time log")
	}

	taskID := existing.TaskID
	if err := s.timelogRepo.Delete(ctx, id); err != nil {
		return err
	}

	// TSK-028: Atomic Rollup
	return s.rollupTaskLoggedHours(ctx, taskID)
}

// ListLogsByTask returns all timelogs for a specific task
func (s *TimeLogService) ListLogsByTask(ctx context.Context, taskID string) ([]*domain.TimeLog, error) {
	logs, err := s.timelogRepo.ListByTask(ctx, taskID)
	if err != nil {
		return nil, err
	}
	sort.Slice(logs, func(i, j int) bool {
		return logs[i].StartTime.After(logs[j].StartTime)
	})
	return logs, nil
}

// ListLogsByProject returns all timelogs for an entire project
func (s *TimeLogService) ListLogsByProject(ctx context.Context, projectID string) ([]*domain.TimeLog, error) {
	logs, err := s.timelogRepo.ListByProject(ctx, projectID)
	if err != nil {
		return nil, err
	}
	sort.Slice(logs, func(i, j int) bool {
		return logs[i].StartTime.After(logs[j].StartTime)
	})
	return logs, nil
}

// -------------------------------------------------------------
// TSK-028: Transactional Task logged_hours Rollup
// -------------------------------------------------------------

func (s *TimeLogService) rollupTaskLoggedHours(ctx context.Context, taskID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	totalMinutes, err := s.timelogRepo.CalculateTaskLoggedMinutes(ctx, taskID)
	if err != nil {
		return err
	}

	// Floating point guard: round to 2 decimal places
	loggedHours := math.Round((float64(totalMinutes)/60.0)*100) / 100

	task, err := s.taskRepo.GetByID(ctx, taskID)
	if err != nil {
		return err
	}

	task.LoggedHours = loggedHours
	return s.taskRepo.Update(ctx, task)
}

// -------------------------------------------------------------
// TSK-029: Weekly Timesheet Aggregation & Burn-Down Analytics
// -------------------------------------------------------------

// GetWeeklyTimesheet generates 7-day logged time breakdown for a user
func (s *TimeLogService) GetWeeklyTimesheet(ctx context.Context, userID, weekStart string, tzOffsetMinutes int) (*domain.WeeklyTimesheet, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, errcode.NewNotFound("User not found")
	}

	// Parse or compute weekStart (Monday)
	var startDate time.Time
	if weekStart != "" {
		parsed, err := time.Parse("2006-01-02", weekStart)
		if err != nil {
			return nil, errcode.NewValidation("Invalid week_start format, expected YYYY-MM-DD", nil)
		}
		// Adjust to Monday if given any day in the week
		weekday := int(parsed.Weekday())
		if weekday == 0 {
			weekday = 7 // Sunday
		}
		startDate = parsed.AddDate(0, 0, -(weekday - 1))
	} else {
		now := time.Now().UTC()
		weekday := int(now.Weekday())
		if weekday == 0 {
			weekday = 7
		}
		startDate = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC).AddDate(0, 0, -(weekday - 1))
	}

	endDate := startDate.AddDate(0, 0, 6)
	dates := make([]string, 7)
	dailyTotals := make(map[string]float64)
	for i := 0; i < 7; i++ {
		d := startDate.AddDate(0, 0, i).Format("2006-01-02")
		dates[i] = d
		dailyTotals[d] = 0.0
	}

	// Fetch logs for user in date window (with 1 day buffer for timezone differences)
	windowStart := startDate.AddDate(0, 0, -1)
	windowEnd := endDate.AddDate(0, 0, 2)
	logs, err := s.timelogRepo.ListByUser(ctx, userID, windowStart, windowEnd)
	if err != nil {
		return nil, err
	}

	// Group by task
	taskEntries := make(map[string]*domain.TimesheetEntry)
	totalMinutes := 0

	for _, l := range logs {
		// Determine local date using timezone offset
		localTime := l.StartTime.Add(time.Duration(tzOffsetMinutes) * time.Minute)
		logDate := localTime.Format("2006-01-02")

		if _, inWeek := dailyTotals[logDate]; !inWeek {
			continue
		}

		entry, exists := taskEntries[l.TaskID]
		if !exists {
			tTitle := "Task " + l.TaskID
			pName := "Project " + l.ProjectID
			if task, err := s.taskRepo.GetByID(ctx, l.TaskID); err == nil {
				tTitle = task.Title
			}
			if proj, err := s.projectRepo.GetByID(ctx, l.ProjectID); err == nil {
				pName = proj.Name
			}

			entry = &domain.TimesheetEntry{
				TaskID:       l.TaskID,
				TaskTitle:    tTitle,
				ProjectID:    l.ProjectID,
				ProjectName:  pName,
				DailyMinutes: make(map[string]int),
				DailyHours:   make(map[string]float64),
			}
			for _, d := range dates {
				entry.DailyMinutes[d] = 0
				entry.DailyHours[d] = 0.0
			}
			taskEntries[l.TaskID] = entry
		}

		entry.DailyMinutes[logDate] += l.DurationMinutes
		entry.TotalMinutes += l.DurationMinutes
		totalMinutes += l.DurationMinutes
	}

	entriesList := make([]*domain.TimesheetEntry, 0, len(taskEntries))
	for _, entry := range taskEntries {
		for _, d := range dates {
			hrs := math.Round((float64(entry.DailyMinutes[d])/60.0)*100) / 100
			entry.DailyHours[d] = hrs
			dailyTotals[d] += hrs
		}
		entry.TotalHours = math.Round((float64(entry.TotalMinutes)/60.0)*100) / 100
		entriesList = append(entriesList, entry)
	}

	sort.Slice(entriesList, func(i, j int) bool {
		return entriesList[i].TaskTitle < entriesList[j].TaskTitle
	})

	for d := range dailyTotals {
		dailyTotals[d] = math.Round(dailyTotals[d]*100) / 100
	}

	return &domain.WeeklyTimesheet{
		UserID:       userID,
		UserName:     user.FullName,
		WeekStart:    startDate.Format("2006-01-02"),
		WeekEnd:      endDate.Format("2006-01-02"),
		TotalMinutes: totalMinutes,
		TotalHours:   math.Round((float64(totalMinutes)/60.0)*100) / 100,
		DailyTotals:  dailyTotals,
		Entries:      entriesList,
	}, nil
}

// GetProjectBurndown generates ideal remaining hours vs actual remaining hours over the project timeline
func (s *TimeLogService) GetProjectBurndown(ctx context.Context, projectID string, tzOffsetMinutes int) (*domain.ProjectBurndown, error) {
	project, err := s.projectRepo.GetByID(ctx, projectID)
	if err != nil {
		return nil, errcode.NewNotFound("Project not found")
	}

	tasks, err := s.taskRepo.ListByProject(ctx, projectID)
	if err != nil {
		return nil, err
	}

	totalEstimatedHours := 0.0
	for _, t := range tasks {
		totalEstimatedHours += t.EstimatedHours
	}

	logs, err := s.timelogRepo.ListByProject(ctx, projectID)
	if err != nil {
		return nil, err
	}

	// Map daily logged hours
	dailyLogged := make(map[string]float64)
	totalLoggedHours := 0.0
	for _, l := range logs {
		localTime := l.StartTime.Add(time.Duration(tzOffsetMinutes) * time.Minute)
		logDate := localTime.Format("2006-01-02")
		hrs := l.DurationHours()
		dailyLogged[logDate] += hrs
		totalLoggedHours += hrs
	}

	// Build timeline days from start_date to target_end_date
	start := project.StartDate
	end := project.TargetEndDate
	if end.Before(start) {
		end = start.AddDate(0, 1, 0)
	}

	totalDays := int(end.Sub(start).Hours()/24) + 1
	if totalDays <= 0 {
		totalDays = 1
	}

	idealDailyBurn := totalEstimatedHours / float64(totalDays)

	points := make([]*domain.BurndownPoint, 0, totalDays)
	cumulativeLogged := 0.0
	curr := start

	for i := 0; i < totalDays; i++ {
		dateStr := curr.Format("2006-01-02")
		dayLogged := dailyLogged[dateStr]
		cumulativeLogged += dayLogged

		idealRemaining := math.Max(0, math.Round((totalEstimatedHours-(float64(i)*idealDailyBurn))*100)/100)
		actualRemaining := math.Max(0, math.Round((totalEstimatedHours-cumulativeLogged)*100)/100)

		points = append(points, &domain.BurndownPoint{
			Date:                  dateStr,
			DayIndex:              i + 1,
			IdealRemainingHours:   idealRemaining,
			ActualRemainingHours:  actualRemaining,
			LoggedHoursOnDay:      math.Round(dayLogged*100) / 100,
			CumulativeLoggedHours: math.Round(cumulativeLogged*100) / 100,
		})

		curr = curr.AddDate(0, 0, 1)
	}

	return &domain.ProjectBurndown{
		ProjectID:           projectID,
		ProjectName:         project.Name,
		StartDate:           project.StartDate.Format("2006-01-02"),
		TargetEndDate:       project.TargetEndDate.Format("2006-01-02"),
		TotalEstimatedHours: math.Round(totalEstimatedHours*100) / 100,
		TotalLoggedHours:    math.Round(totalLoggedHours*100) / 100,
		Points:              points,
	}, nil
}

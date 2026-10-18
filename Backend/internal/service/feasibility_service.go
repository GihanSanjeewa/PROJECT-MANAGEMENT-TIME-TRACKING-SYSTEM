package service

import (
	"context"
	"fmt"
	"math"
	"time"

	"project-management-system/internal/domain"
	"project-management-system/internal/repository"
	"project-management-system/pkg/errcode"
)

type FeasibilityService struct {
	projectRepo     repository.ProjectRepository
	taskRepo        repository.TaskRepository
	userRepo        repository.UserRepository
	calendarService *CalendarService
}

func NewFeasibilityService(
	projectRepo repository.ProjectRepository,
	taskRepo repository.TaskRepository,
	userRepo repository.UserRepository,
	calendarService *CalendarService,
) *FeasibilityService {
	return &FeasibilityService{
		projectRepo:     projectRepo,
		taskRepo:        taskRepo,
		userRepo:        userRepo,
		calendarService: calendarService,
	}
}

// EvaluateProject calculates comprehensive deadline feasibility (TSK-022, TSK-023, TSK-024)
func (s *FeasibilityService) EvaluateProject(ctx context.Context, projectID string) (*domain.FeasibilityReport, error) {
	project, err := s.projectRepo.GetByID(ctx, projectID)
	if err != nil {
		return nil, errcode.NewNotFound("Project not found")
	}

	members, err := s.projectRepo.GetMembers(ctx, projectID)
	if err != nil {
		return nil, err
	}

	tasks, err := s.taskRepo.ListByProject(ctx, projectID)
	if err != nil {
		return nil, err
	}

	deps, err := s.taskRepo.GetDependencies(ctx, projectID)
	if err != nil {
		deps = make([]*domain.TaskDependency, 0)
	}

	holidays, err := s.projectRepo.GetHolidays(ctx)
	if err != nil {
		holidays = make([]*domain.CompanyHoliday, 0)
	}

	now := time.Now().UTC()
	return s.evaluateInternal(project, members, tasks, deps, holidays, now, 0.0)
}

// evaluateInternal is a pure mathematical calculation engine supporting both real evaluation and What-If simulation
func (s *FeasibilityService) evaluateInternal(
	project *domain.Project,
	members []*domain.ProjectMember,
	tasks []*domain.Task,
	deps []*domain.TaskDependency,
	holidays []*domain.CompanyHoliday,
	evalDate time.Time,
	overtimeHoursPerDay float64,
) (*domain.FeasibilityReport, error) {
	evalStart := time.Date(evalDate.Year(), evalDate.Month(), evalDate.Day(), 0, 0, 0, 0, time.UTC)
	projStart := time.Date(project.StartDate.Year(), project.StartDate.Month(), project.StartDate.Day(), 0, 0, 0, 0, time.UTC)
	projEnd := time.Date(project.TargetEndDate.Year(), project.TargetEndDate.Month(), project.TargetEndDate.Day(), 0, 0, 0, 0, time.UTC)

	if evalStart.Before(projStart) {
		evalStart = projStart
	}

	remainingWorkingDays := s.calendarService.CalculateWorkingDays(evalStart, projEnd, holidays)

	// 1. TSK-022: Role-Level Remaining Effort & Incomplete Task Aggregation
	roleRemainingHours := map[string]float64{
		"DEVELOPER": 0.0,
		"QA":        0.0,
		"BA":        0.0,
	}
	totalRemainingHours := 0.0
	incompleteTasks := make([]*domain.Task, 0)

	for _, t := range tasks {
		if t.Status == domain.TaskStatusDone {
			continue
		}
		rem := math.Max(0, t.EstimatedHours-t.LoggedHours)
		roleKey := string(t.RequiredRole)
		roleRemainingHours[roleKey] += rem
		totalRemainingHours += rem
		incompleteTasks = append(incompleteTasks, t)
	}

	// 2. TSK-022: Role-Level & Individual Member Available Capacity
	roleCapacityHours := map[string]float64{
		"DEVELOPER": 0.0,
		"QA":        0.0,
		"BA":        0.0,
		"PM":        0.0,
	}
	roleMemberCounts := map[string]int{
		"DEVELOPER": 0,
		"QA":        0,
		"BA":        0,
		"PM":        0,
	}
	roleDailyVelocity := map[string]float64{
		"DEVELOPER": 0.0,
		"QA":        0.0,
		"BA":        0.0,
	}

	memberCapacityMap := make(map[string]float64)
	memberDailyMap := make(map[string]float64)
	totalCapacityHours := 0.0

	for _, m := range members {
		roleKey := string(m.ProjectRole)
		dailyCap := m.EffectiveDailyHours + overtimeHoursPerDay
		if dailyCap <= 0 {
			dailyCap = (8.0 * (float64(m.AllocationPct) / 100.0)) + overtimeHoursPerDay
		}

		mStart := evalStart
		if !m.JoinedDate.IsZero() && m.JoinedDate.After(mStart) {
			mStart = m.JoinedDate
		}
		mEnd := projEnd
		if m.LeftDate != nil && m.LeftDate.Before(mEnd) {
			mEnd = *m.LeftDate
		}

		mWorkingDays := s.calendarService.CalculateWorkingDays(mStart, mEnd, holidays)
		mTotalCap := float64(mWorkingDays) * dailyCap

		roleCapacityHours[roleKey] += mTotalCap
		roleMemberCounts[roleKey]++
		roleDailyVelocity[roleKey] += dailyCap
		memberCapacityMap[m.UserID] += mTotalCap
		memberDailyMap[m.UserID] = dailyCap

		if roleKey != "PM" {
			totalCapacityHours += mTotalCap
		}
	}

	// Calculate Role Metrics
	roleMetrics := make(map[string]domain.RoleMetric)
	canMeetRoles := true

	for _, roleKey := range []string{"DEVELOPER", "QA", "BA"} {
		avail := math.Round(roleCapacityHours[roleKey]*100) / 100
		rem := math.Round(roleRemainingHours[roleKey]*100) / 100
		buf := math.Round((avail-rem)*100) / 100

		utilization := 0.0
		if avail > 0 {
			utilization = math.Round((rem/avail)*1000) / 10
		} else if rem > 0 {
			utilization = 999.0
		}

		status := domain.RoleStatusOnTrack
		if (avail == 0 && rem > 0) || buf < 0 {
			status = domain.RoleStatusCannotMeet
			canMeetRoles = false
		} else if utilization > 85.0 {
			status = domain.RoleStatusAtRisk
		}

		roleMetrics[roleKey] = domain.RoleMetric{
			Role:                   roleKey,
			MemberCount:            roleMemberCounts[roleKey],
			AvailableCapacityHours: avail,
			RemainingTaskHours:     rem,
			BufferOrDeficitHours:   buf,
			UtilizationPct:         utilization,
			Status:                 status,
		}
	}

	// 3. TSK-023: Individual Developer Bottleneck Analyzer
	bottlenecks := make([]domain.MemberBottleneck, 0)
	memberAssignedHours := make(map[string]float64)

	for _, t := range incompleteTasks {
		if t.AssignedTo != nil && *t.AssignedTo != "" {
			rem := math.Max(0, t.EstimatedHours-t.LoggedHours)
			memberAssignedHours[*t.AssignedTo] += rem
		}
	}

	hasSevereBottleneck := false
	for _, m := range members {
		assignedHrs := memberAssignedHours[m.UserID]
		availHrs := memberCapacityMap[m.UserID]
		if assignedHrs > availHrs {
			hasSevereBottleneck = true
			deficit := math.Round((assignedHrs-availHrs)*100) / 100
			bottlenecks = append(bottlenecks, domain.MemberBottleneck{
				UserID:                 m.UserID,
				Name:                   m.UserID,
				Role:                   string(m.ProjectRole),
				AllocatedTaskHours:     math.Round(assignedHrs*100) / 100,
				AvailableCapacityHours: math.Round(availHrs*100) / 100,
				DeficitHours:           deficit,
				IsOverloaded:           true,
				Recommendation:         fmt.Sprintf("Reassign %vh of tasks from %s to underutilized %s team members", deficit, m.UserID, m.ProjectRole),
			})
		}
	}

	// 4. TSK-024: Critical Path Method (CPM) & Projected Completion Date
	criticalPath, projectDurationDays := s.calculateCriticalPath(incompleteTasks, deps, roleDailyVelocity)
	projectedCompletionDate := s.calendarService.AddWorkingDays(evalStart, projectDurationDays, holidays)

	daysAheadOrBehind := 0
	if projectedCompletionDate.After(projEnd) {
		daysAheadOrBehind = -s.calendarService.CalculateWorkingDays(projEnd, projectedCompletionDate, holidays)
	} else {
		daysAheadOrBehind = s.calendarService.CalculateWorkingDays(projectedCompletionDate, projEnd, holidays)
	}

	// 5. Overall Feasibility Score & Status
	totalBufferHours := math.Round((totalCapacityHours-totalRemainingHours)*100) / 100
	canMeetDeadline := canMeetRoles && (daysAheadOrBehind >= 0)

	feasibilityScore := 100.0
	if totalRemainingHours > 0 {
		feasibilityScore = math.Min(100.0, (totalCapacityHours/totalRemainingHours)*100.0)
		if daysAheadOrBehind < 0 {
			// Penalty for projected slippage
			slippagePenalty := float64(-daysAheadOrBehind) * 8.0
			feasibilityScore = math.Max(5.0, feasibilityScore-slippagePenalty)
		}
	}
	feasibilityScore = math.Round(feasibilityScore*10) / 10

	feasibilityStatus := domain.StatusOnTrack
	if !canMeetDeadline {
		feasibilityStatus = domain.StatusCriticalDelay
	} else if hasSevereBottleneck || daysAheadOrBehind <= 2 || len(bottlenecks) > 0 {
		feasibilityStatus = domain.StatusAtRisk
	}

	return &domain.FeasibilityReport{
		ProjectID:               project.ID,
		ProjectName:             project.Name,
		TargetEndDate:           project.TargetEndDate,
		RemainingWorkingDays:    remainingWorkingDays,
		CanMeetDeadline:         canMeetDeadline,
		FeasibilityScore:        feasibilityScore,
		FeasibilityStatus:       feasibilityStatus,
		ProjectedCompletionDate: projectedCompletionDate,
		DaysAheadOrBehind:       daysAheadOrBehind,
		TotalRemainingHours:     math.Round(totalRemainingHours*100) / 100,
		TotalCapacityHours:      math.Round(totalCapacityHours*100) / 100,
		TotalBufferHours:        totalBufferHours,
		CriticalPath:            criticalPath,
		Roles:                   roleMetrics,
		IndividualBottlenecks:   bottlenecks,
		CalculatedAt:            time.Now().UTC(),
	}, nil
}

// calculateCriticalPath computes the CPM forward/backward pass over incomplete tasks
func (s *FeasibilityService) calculateCriticalPath(
	tasks []*domain.Task,
	deps []*domain.TaskDependency,
	roleDailyVelocity map[string]float64,
) ([]string, int) {
	if len(tasks) == 0 {
		return []string{}, 0
	}

	taskMap := make(map[string]*domain.Task)
	taskDuration := make(map[string]int)

	for _, t := range tasks {
		taskMap[t.ID] = t
		rem := math.Max(0, t.EstimatedHours-t.LoggedHours)
		vel := roleDailyVelocity[string(t.RequiredRole)]
		if vel <= 0 {
			vel = 8.0 // fallback
		}
		duration := int(math.Ceil(rem / vel))
		if duration < 1 && rem > 0 {
			duration = 1
		}
		taskDuration[t.ID] = duration
	}

	// Adjacency graph for dependencies among incomplete tasks
	// Edge: U -> V means V depends on U (U is predecessor of V)
	successors := make(map[string][]string)
	predecessors := make(map[string][]string)

	for _, d := range deps {
		if _, ok1 := taskMap[d.TaskID]; ok1 {
			if _, ok2 := taskMap[d.DependsOnTaskID]; ok2 {
				successors[d.DependsOnTaskID] = append(successors[d.DependsOnTaskID], d.TaskID)
				predecessors[d.TaskID] = append(predecessors[d.TaskID], d.DependsOnTaskID)
			}
		}
	}

	// Forward Pass: Earliest Start (ES) and Earliest Finish (EF)
	es := make(map[string]int)
	ef := make(map[string]int)

	// In-degree for topological order
	inDegree := make(map[string]int)
	for id := range taskMap {
		inDegree[id] = len(predecessors[id])
	}

	queue := make([]string, 0)
	for id, deg := range inDegree {
		if deg == 0 {
			queue = append(queue, id)
			es[id] = 0
			ef[id] = taskDuration[id]
		}
	}

	maxProjectDays := 0
	topoOrder := make([]string, 0)

	for len(queue) > 0 {
		curr := queue[0]
		queue = queue[1:]
		topoOrder = append(topoOrder, curr)

		if ef[curr] > maxProjectDays {
			maxProjectDays = ef[curr]
		}

		for _, succ := range successors[curr] {
			if ef[curr] > es[succ] {
				es[succ] = ef[curr]
				ef[succ] = es[succ] + taskDuration[succ]
			}
			inDegree[succ]--
			if inDegree[succ] == 0 {
				queue = append(queue, succ)
			}
		}
	}

	// Backward Pass: Latest Start (LS) and Latest Finish (LF)
	ls := make(map[string]int)
	lf := make(map[string]int)

	for id := range taskMap {
		lf[id] = maxProjectDays
		ls[id] = maxProjectDays - taskDuration[id]
	}

	// Traverse reverse topological order
	for i := len(topoOrder) - 1; i >= 0; i-- {
		curr := topoOrder[i]
		for _, succ := range successors[curr] {
			if ls[succ] < lf[curr] {
				lf[curr] = ls[succ]
				ls[curr] = lf[curr] - taskDuration[curr]
			}
		}
	}

	// Critical Path = tasks with zero slack (LS == ES) and duration > 0
	criticalPath := make([]string, 0)
	for _, id := range topoOrder {
		slack := ls[id] - es[id]
		if slack == 0 && taskDuration[id] > 0 {
			criticalPath = append(criticalPath, id)
		}
	}

	return criticalPath, maxProjectDays
}

// SimulateWhatIf implements TSK-025: In-Memory What-If Simulation
func (s *FeasibilityService) SimulateWhatIf(
	ctx context.Context,
	projectID string,
	req domain.SimulationRequest,
) (*domain.SimulationResult, error) {
	// Baseline evaluation
	origReport, err := s.EvaluateProject(ctx, projectID)
	if err != nil {
		return nil, err
	}

	project, err := s.projectRepo.GetByID(ctx, projectID)
	if err != nil {
		return nil, err
	}

	members, err := s.projectRepo.GetMembers(ctx, projectID)
	if err != nil {
		return nil, err
	}

	tasks, err := s.taskRepo.ListByProject(ctx, projectID)
	if err != nil {
		return nil, err
	}

	deps, err := s.taskRepo.GetDependencies(ctx, projectID)
	if err != nil {
		deps = make([]*domain.TaskDependency, 0)
	}

	holidays, err := s.projectRepo.GetHolidays(ctx)
	if err != nil {
		holidays = make([]*domain.CompanyHoliday, 0)
	}

	// Clone project for hypothetical simulation (zero database mutation)
	simProj := *project
	if req.ExtendDeadlineDays > 0 {
		simProj.TargetEndDate = simProj.TargetEndDate.AddDate(0, 0, req.ExtendDeadlineDays)
	}

	// Clone and add hypothetical members
	simMembers := make([]*domain.ProjectMember, len(members))
	copy(simMembers, members)

	now := time.Now().UTC()
	for i := 0; i < req.AdditionalDevs; i++ {
		simMembers = append(simMembers, &domain.ProjectMember{
			ID:                  fmt.Sprintf("sim-dev-%d", i+1),
			ProjectID:           projectID,
			UserID:              fmt.Sprintf("Simulated Developer %d", i+1),
			ProjectRole:         domain.ProjectRoleDeveloper,
			AllocationPct:       100,
			EffectiveDailyHours: 8.0,
			JoinedDate:          now,
		})
	}

	for i := 0; i < req.AdditionalQA; i++ {
		simMembers = append(simMembers, &domain.ProjectMember{
			ID:                  fmt.Sprintf("sim-qa-%d", i+1),
			ProjectID:           projectID,
			UserID:              fmt.Sprintf("Simulated QA %d", i+1),
			ProjectRole:         domain.ProjectRoleQA,
			AllocationPct:       100,
			EffectiveDailyHours: 8.0,
			JoinedDate:          now,
		})
	}

	for i := 0; i < req.AdditionalBA; i++ {
		simMembers = append(simMembers, &domain.ProjectMember{
			ID:                  fmt.Sprintf("sim-ba-%d", i+1),
			ProjectID:           projectID,
			UserID:              fmt.Sprintf("Simulated BA %d", i+1),
			ProjectRole:         domain.ProjectRoleBA,
			AllocationPct:       100,
			EffectiveDailyHours: 8.0,
			JoinedDate:          now,
		})
	}

	simReport, err := s.evaluateInternal(
		&simProj,
		simMembers,
		tasks,
		deps,
		holidays,
		now,
		req.OvertimeHoursPerDay,
	)
	if err != nil {
		return nil, err
	}

	// Build human-friendly recommendation summary
	recText := "Simulation completed. "
	if !origReport.CanMeetDeadline && simReport.CanMeetDeadline {
		recText += fmt.Sprintf("Success! Proposed staffing and timeline adjustments turn project ON_TRACK, boosting score from %v%% to %v%%.",
			origReport.FeasibilityScore, simReport.FeasibilityScore)
	} else if simReport.FeasibilityScore > origReport.FeasibilityScore {
		recText += fmt.Sprintf("Feasibility improved by +%v%% (buffer: %v days).",
			math.Round((simReport.FeasibilityScore-origReport.FeasibilityScore)*10)/10, simReport.DaysAheadOrBehind)
	} else {
		recText += "Adjustments have minimal impact on the critical path."
	}

	return &domain.SimulationResult{
		OriginalCanMeetDeadline:  origReport.CanMeetDeadline,
		SimulatedCanMeetDeadline: simReport.CanMeetDeadline,
		OriginalScore:            origReport.FeasibilityScore,
		SimulatedScore:           simReport.FeasibilityScore,
		OriginalProjectedDate:    origReport.ProjectedCompletionDate,
		SimulatedProjectedDate:   simReport.ProjectedCompletionDate,
		OriginalDaysBuffer:       origReport.DaysAheadOrBehind,
		SimulatedDaysBuffer:      simReport.DaysAheadOrBehind,
		OriginalReport:           *origReport,
		SimulatedReport:          *simReport,
		RecommendationSummary:    recText,
	}, nil
}

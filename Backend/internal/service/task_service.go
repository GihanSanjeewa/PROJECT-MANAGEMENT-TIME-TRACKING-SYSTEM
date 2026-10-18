package service

import (
	"context"
	"fmt"
	"math"
	"strings"
	"sync/atomic"
	"time"

	"project-management-system/internal/domain"
	"project-management-system/internal/repository"
	"project-management-system/pkg/errcode"
)

var taskSeqNum int64

type TaskService struct {
	taskRepo        repository.TaskRepository
	projectRepo     repository.ProjectRepository
	userRepo        repository.UserRepository
	calendarService *CalendarService
}

func NewTaskService(
	taskRepo repository.TaskRepository,
	projectRepo repository.ProjectRepository,
	userRepo repository.UserRepository,
	calendarService *CalendarService,
) *TaskService {
	return &TaskService{
		taskRepo:        taskRepo,
		projectRepo:     projectRepo,
		userRepo:        userRepo,
		calendarService: calendarService,
	}
}

// CreateTask creates a task enforcing timeline and membership invariants (TSK-017, TSK-018)
func (s *TaskService) CreateTask(ctx context.Context, task *domain.Task) (*domain.Task, error) {
	if err := task.Validate(); err != nil {
		return nil, err
	}

	// 1. Fetch parent project and verify timeline range (TSK-017)
	project, err := s.projectRepo.GetByID(ctx, task.ProjectID)
	if err != nil {
		return nil, errcode.NewNotFound("Project not found")
	}

	if err := task.AssertWithinProjectTimeline(project); err != nil {
		return nil, err
	}

	// 2. Validate Assignee membership & role (TSK-018)
	if task.AssignedTo != nil && *task.AssignedTo != "" {
		if err := s.validateAssignee(ctx, project.ID, *task.AssignedTo, task.RequiredRole, task.StartDate, task.DueDate); err != nil {
			return nil, err
		}
	}

	// Default status if empty
	if task.Status == "" {
		task.Status = domain.TaskStatusTodo
	}
	if task.Priority == "" {
		task.Priority = domain.PriorityMedium
	}
	if task.ID == "" {
		task.ID = fmt.Sprintf("task-%d-%d", time.Now().UnixNano(), atomic.AddInt64(&taskSeqNum, 1))
	}

	if err := s.taskRepo.Create(ctx, task); err != nil {
		return nil, err
	}

	return task, nil
}

// UpdateTask updates task details enforcing timeline and membership invariants
func (s *TaskService) UpdateTask(ctx context.Context, taskID string, update *domain.Task) (*domain.Task, error) {
	existing, err := s.taskRepo.GetByID(ctx, taskID)
	if err != nil {
		return nil, err
	}

	project, err := s.projectRepo.GetByID(ctx, existing.ProjectID)
	if err != nil {
		return nil, errcode.NewNotFound("Project not found")
	}

	// Merge fields
	if update.Title != "" {
		existing.Title = update.Title
	}
	if update.Description != "" {
		existing.Description = update.Description
	}
	if update.RequiredRole.IsValid() {
		existing.RequiredRole = update.RequiredRole
	}
	if update.Priority.IsValid() {
		existing.Priority = update.Priority
	}
	if update.EstimatedHours > 0 {
		existing.EstimatedHours = update.EstimatedHours
	}
	if !update.StartDate.IsZero() {
		existing.StartDate = update.StartDate
	}
	if !update.DueDate.IsZero() {
		existing.DueDate = update.DueDate
	}
	if update.AssignedTo != nil {
		existing.AssignedTo = update.AssignedTo
	}

	// Invariant validation
	if err := existing.Validate(); err != nil {
		return nil, err
	}
	if err := existing.AssertWithinProjectTimeline(project); err != nil {
		return nil, err
	}

	if existing.AssignedTo != nil && *existing.AssignedTo != "" {
		if err := s.validateAssignee(ctx, project.ID, *existing.AssignedTo, existing.RequiredRole, existing.StartDate, existing.DueDate); err != nil {
			return nil, err
		}
	}

	if err := s.taskRepo.Update(ctx, existing); err != nil {
		return nil, err
	}

	return existing, nil
}

// GetTaskByID retrieves task by ID
func (s *TaskService) GetTaskByID(ctx context.Context, id string) (*domain.Task, error) {
	return s.taskRepo.GetByID(ctx, id)
}

// ListTasksByProject retrieves all tasks in a project
func (s *TaskService) ListTasksByProject(ctx context.Context, projectID string) ([]*domain.Task, error) {
	return s.taskRepo.ListByProject(ctx, projectID)
}

// DeleteTask removes a task
func (s *TaskService) DeleteTask(ctx context.Context, id string) error {
	return s.taskRepo.Delete(ctx, id)
}

// validateAssignee enforces TSK-018
func (s *TaskService) validateAssignee(
	ctx context.Context,
	projectID, userID string,
	requiredRole domain.RequiredRole,
	taskStart, taskDue time.Time,
) error {
	member, err := s.projectRepo.GetMember(ctx, projectID, userID)
	if err != nil {
		return errcode.New(
			errcode.ErrValidationFailed,
			fmt.Sprintf("Assignee (%s) is not an assigned team member of this project", userID),
			422,
			map[string]string{"user_id": userID, "project_id": projectID},
		)
	}

	// Check member active window
	if !member.JoinedDate.IsZero() && taskStart.Before(member.JoinedDate) {
		return errcode.New(
			errcode.ErrValidationFailed,
			fmt.Sprintf("Task starts (%s) before team member's joined date (%s)",
				taskStart.Format("2006-01-02"), member.JoinedDate.Format("2006-01-02")),
			422,
			nil,
		)
	}

	if member.LeftDate != nil && taskDue.After(*member.LeftDate) {
		return errcode.New(
			errcode.ErrValidationFailed,
			fmt.Sprintf("Task due date (%s) exceeds team member's scheduled departure date (%s)",
				taskDue.Format("2006-01-02"), member.LeftDate.Format("2006-01-02")),
			422,
			nil,
		)
	}

	// Role compatibility check (unless PM is assigning or member has matching functional role)
	memberRole := strings.ToUpper(string(member.ProjectRole))
	reqRole := strings.ToUpper(string(requiredRole))
	if memberRole != reqRole && memberRole != "PM" {
		return errcode.New(
			errcode.ErrValidationFailed,
			fmt.Sprintf("Member project role (%s) does not match task required role (%s)",
				member.ProjectRole, requiredRole),
			422,
			map[string]string{"member_role": string(member.ProjectRole), "required_role": string(requiredRole)},
		)
	}

	return nil
}

// AddDependency implements TSK-019: Finish-to-Start dependency with DAG Cycle Detection
func (s *TaskService) AddDependency(
	ctx context.Context,
	projectID, taskID, dependsOnTaskID, depType string,
) (*domain.TaskDependency, error) {
	// 1. Block self-dependency
	if taskID == dependsOnTaskID {
		return nil, errcode.New(
			errcode.ErrValidationFailed,
			"Self-dependency is disallowed: a task cannot depend on itself",
			422,
			nil,
		)
	}

	// 2. Fetch both tasks and ensure they belong to the same project
	task, err := s.taskRepo.GetByID(ctx, taskID)
	if err != nil {
		return nil, errcode.NewNotFound(fmt.Sprintf("Task not found: %s", taskID))
	}

	predecessor, err := s.taskRepo.GetByID(ctx, dependsOnTaskID)
	if err != nil {
		return nil, errcode.NewNotFound(fmt.Sprintf("Predecessor task not found: %s", dependsOnTaskID))
	}

	if task.ProjectID != projectID || predecessor.ProjectID != projectID {
		return nil, errcode.New(
			errcode.ErrValidationFailed,
			"Cross-project dependencies are disallowed: both tasks must belong to the same project",
			422,
			nil,
		)
	}

	// 3. Finish-to-Start timing rule: successor.StartDate >= predecessor.DueDate
	taskStart := time.Date(task.StartDate.Year(), task.StartDate.Month(), task.StartDate.Day(), 0, 0, 0, 0, time.UTC)
	predDue := time.Date(predecessor.DueDate.Year(), predecessor.DueDate.Month(), predecessor.DueDate.Day(), 0, 0, 0, 0, time.UTC)
	if taskStart.Before(predDue) {
		return nil, errcode.New(
			errcode.ErrValidationFailed,
			fmt.Sprintf("Finish-to-Start violation: Successor task starts on %s, before predecessor due date %s",
				taskStart.Format("2006-01-02"), predDue.Format("2006-01-02")),
			422,
			map[string]string{
				"task_start":      taskStart.Format("2006-01-02"),
				"predecessor_due": predDue.Format("2006-01-02"),
			},
		)
	}

	// 4. DAG Cycle Detection using DFS
	// If predecessor is reachable from task, then adding edge (dependsOnTaskID -> taskID) creates a cycle!
	existingDeps, err := s.taskRepo.GetDependencies(ctx, projectID)
	if err != nil {
		return nil, err
	}

	// Build adjacency list: node -> outgoing edges (nodes that depend on this node)
	// Edge: U -> V means V depends on U
	adj := make(map[string][]string)
	for _, d := range existingDeps {
		adj[d.DependsOnTaskID] = append(adj[d.DependsOnTaskID], d.TaskID)
	}

	// If there is already a path from taskID to dependsOnTaskID, adding dependsOnTaskID -> taskID creates a cycle
	if hasPath(taskID, dependsOnTaskID, adj) {
		return nil, errcode.New(
			errcode.ErrValidationFailed,
			fmt.Sprintf("Cycle detected: adding dependency (%s -> %s) would create a circular dependency chain",
				dependsOnTaskID, taskID),
			422,
			map[string]string{"task_id": taskID, "depends_on_task_id": dependsOnTaskID},
		)
	}

	if depType == "" {
		depType = "FS"
	}

	dep := &domain.TaskDependency{
		ID:              fmt.Sprintf("dep-%d-%d", time.Now().UnixNano(), atomic.AddInt64(&taskSeqNum, 1)),
		ProjectID:       projectID,
		TaskID:          taskID,
		DependsOnTaskID: dependsOnTaskID,
		DependencyType:  depType,
	}

	if err := s.taskRepo.AddDependency(ctx, dep); err != nil {
		return nil, err
	}

	return dep, nil
}

// hasPath performs BFS to verify reachability in DAG
func hasPath(start, target string, adj map[string][]string) bool {
	visited := make(map[string]bool)
	queue := []string{start}
	visited[start] = true

	for len(queue) > 0 {
		curr := queue[0]
		queue = queue[1:]

		if curr == target {
			return true
		}

		for _, neighbor := range adj[curr] {
			if !visited[neighbor] {
				visited[neighbor] = true
				queue = append(queue, neighbor)
			}
		}
	}
	return false
}

// GetProjectDependencies retrieves all dependencies in a project
func (s *TaskService) GetProjectDependencies(ctx context.Context, projectID string) ([]*domain.TaskDependency, error) {
	return s.taskRepo.GetDependencies(ctx, projectID)
}

// RemoveDependency deletes a dependency link
func (s *TaskService) RemoveDependency(ctx context.Context, id string) error {
	return s.taskRepo.DeleteDependency(ctx, id)
}

// UpdateTaskStatus implements TSK-021: State Machine & Hand-off Workflow
func (s *TaskService) UpdateTaskStatus(
	ctx context.Context,
	taskID string,
	newStatus domain.TaskStatus,
	changedByUserID, userRole, comment string,
) (*domain.Task, *domain.TaskAuditLog, error) {
	task, err := s.taskRepo.GetByID(ctx, taskID)
	if err != nil {
		return nil, nil, err
	}

	// Check if task has any open subtasks
	subtasks, err := s.taskRepo.GetSubtasks(ctx, taskID)
	if err != nil {
		return nil, nil, err
	}

	hasOpenSubtasks := false
	for _, sub := range subtasks {
		if sub.Status != domain.TaskStatusDone {
			hasOpenSubtasks = true
			break
		}
	}

	// Validate status transition
	oldStatus := task.Status
	if err := domain.ValidateStatusTransition(oldStatus, newStatus, userRole, hasOpenSubtasks); err != nil {
		return nil, nil, err
	}

	task.Status = newStatus
	if err := s.taskRepo.Update(ctx, task); err != nil {
		return nil, nil, err
	}

	// Create audit log entry
	audit := &domain.TaskAuditLog{
		ID:        fmt.Sprintf("audit-%d-%d", time.Now().UnixNano(), atomic.AddInt64(&taskSeqNum, 1)),
		TaskID:    taskID,
		ChangedBy: changedByUserID,
		OldStatus: oldStatus,
		NewStatus: newStatus,
		Comment:   comment,
		CreatedAt: time.Now().UTC(),
	}

	_ = s.taskRepo.AddAuditLog(ctx, audit)

	return task, audit, nil
}

// GetTaskAuditLogs retrieves history of status transitions
func (s *TaskService) GetTaskAuditLogs(ctx context.Context, taskID string) ([]*domain.TaskAuditLog, error) {
	return s.taskRepo.GetAuditLogs(ctx, taskID)
}

// GetProjectWorkload implements TSK-020: Developer Daily Workload & Over-Allocation Detector
func (s *TaskService) GetProjectWorkload(ctx context.Context, projectID string) ([]*domain.AssigneeWorkload, error) {
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

	holidays, err := s.projectRepo.GetHolidays(ctx)
	if err != nil {
		holidays = make([]*domain.CompanyHoliday, 0)
	}

	workloads := make([]*domain.AssigneeWorkload, 0)

	for _, m := range members {
		// Daily capacity from member allocation
		dailyCap := m.EffectiveDailyHours
		if dailyCap <= 0 {
			dailyCap = 8.0 * (float64(m.AllocationPct) / 100.0)
		}

		user, _ := s.userRepo.GetByID(ctx, m.UserID)
		userName := m.UserID
		if user != nil {
			userName = user.FullName
		}

		// Collect tasks for this user
		memberTasks := make([]*domain.Task, 0)
		totalHours := 0.0
		for _, t := range tasks {
			if t.AssignedTo != nil && *t.AssignedTo == m.UserID && t.Status != domain.TaskStatusDone {
				memberTasks = append(memberTasks, t)
				totalHours += t.EstimatedHours
			}
		}

		// Calculate daily workload map: dateStr -> totalHours
		dailyMap := make(map[string]float64)

		for _, t := range memberTasks {
			workingDays := s.calendarService.CalculateWorkingDays(t.StartDate, t.DueDate, holidays)
			if workingDays <= 0 {
				workingDays = 1
			}
			hoursPerDay := t.EstimatedHours / float64(workingDays)

			// Spread hours across working days
			curr := t.StartDate
			for !curr.After(t.DueDate) {
				if s.calendarService.IsWorkingDay(curr, holidays) {
					dateKey := curr.Format("2006-01-02")
					dailyMap[dateKey] += hoursPerDay
				}
				curr = curr.AddDate(0, 0, 1)
			}
		}

		// Construct breakdown across project timeline
		breakdown := make([]domain.WorkloadDay, 0)
		isOverallocated := false
		maxDailyHours := 0.0

		currDate := project.StartDate
		for !currDate.After(project.TargetEndDate) {
			if s.calendarService.IsWorkingDay(currDate, holidays) {
				dateStr := currDate.Format("2006-01-02")
				dayHours := dailyMap[dateStr]
				isOver := dayHours > dailyCap
				excess := 0.0
				if isOver {
					excess = math.Round((dayHours-dailyCap)*100) / 100
					isOverallocated = true
				}
				if dayHours > maxDailyHours {
					maxDailyHours = dayHours
				}

				breakdown = append(breakdown, domain.WorkloadDay{
					Date:         dateStr,
					Hours:        math.Round(dayHours*100) / 100,
					CapacityHrs:  dailyCap,
					IsOverloaded: isOver,
					ExcessHours:  excess,
				})
			}
			currDate = currDate.AddDate(0, 0, 1)
		}

		workloads = append(workloads, &domain.AssigneeWorkload{
			UserID:           m.UserID,
			UserName:         userName,
			Role:             string(m.ProjectRole),
			DailyCapacity:    dailyCap,
			TotalAssignedHrs: totalHours,
			IsOverallocated:  isOverallocated,
			MaxDailyHours:    math.Round(maxDailyHours*100) / 100,
			DailyBreakdown:   breakdown,
		})
	}

	return workloads, nil
}

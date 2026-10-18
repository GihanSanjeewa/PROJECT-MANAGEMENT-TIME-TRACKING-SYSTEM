package domain

import (
	"fmt"
	"strings"
	"time"

	"project-management-system/pkg/errcode"
)

type TaskStatus string

const (
	TaskStatusBacklog    TaskStatus = "BACKLOG"
	TaskStatusTodo       TaskStatus = "TODO"
	TaskStatusInProgress TaskStatus = "IN_PROGRESS"
	TaskStatusCodeReview TaskStatus = "CODE_REVIEW"
	TaskStatusInQA       TaskStatus = "IN_QA"
	TaskStatusDone       TaskStatus = "DONE"
	TaskStatusBlocked    TaskStatus = "BLOCKED"
)

func (s TaskStatus) IsValid() bool {
	switch s {
	case TaskStatusBacklog, TaskStatusTodo, TaskStatusInProgress, TaskStatusCodeReview, TaskStatusInQA, TaskStatusDone, TaskStatusBlocked:
		return true
	default:
		return false
	}
}

type Priority string

const (
	PriorityLow    Priority = "LOW"
	PriorityMedium Priority = "MEDIUM"
	PriorityHigh   Priority = "HIGH"
	PriorityUrgent Priority = "URGENT"
)

func (p Priority) IsValid() bool {
	switch p {
	case PriorityLow, PriorityMedium, PriorityHigh, PriorityUrgent:
		return true
	default:
		return false
	}
}

type RequiredRole string

const (
	RoleRequiredDev RequiredRole = "DEVELOPER"
	RoleRequiredQA  RequiredRole = "QA"
	RoleRequiredBA  RequiredRole = "BA"
)

func (r RequiredRole) IsValid() bool {
	switch r {
	case RoleRequiredDev, RoleRequiredQA, RoleRequiredBA:
		return true
	default:
		return false
	}
}

type Task struct {
	ID             string       `json:"id"`
	ProjectID      string       `json:"project_id"`
	ParentTaskID   *string      `json:"parent_task_id,omitempty"`
	Title          string       `json:"title"`
	Description    string       `json:"description"`
	AssignedTo     *string      `json:"assigned_to,omitempty"`
	RequiredRole   RequiredRole `json:"required_role"`
	Status         TaskStatus   `json:"status"`
	Priority       Priority     `json:"priority"`
	EstimatedHours float64      `json:"estimated_hours"`
	LoggedHours    float64      `json:"logged_hours"`
	StartDate      time.Time    `json:"start_date"`
	DueDate        time.Time    `json:"due_date"`
	CreatedAt      time.Time    `json:"created_at"`
	UpdatedAt      time.Time    `json:"updated_at"`
}

func (t *Task) Validate() error {
	if strings.TrimSpace(t.Title) == "" {
		return errcode.NewValidation("Task title cannot be blank", nil)
	}
	if !t.RequiredRole.IsValid() {
		return errcode.NewValidation("Required role must be one of: DEVELOPER, QA, BA", nil)
	}
	if t.EstimatedHours <= 0 {
		return errcode.NewValidation("Estimated hours must be greater than zero", nil)
	}
	if t.LoggedHours < 0 {
		return errcode.NewValidation("Logged hours cannot be negative", nil)
	}
	if t.DueDate.Before(t.StartDate) {
		return errcode.NewValidation("Task due date must be on or after start date", nil)
	}
	if t.Status != "" && !t.Status.IsValid() {
		return errcode.NewValidation("Invalid task status", nil)
	}
	if t.Priority != "" && !t.Priority.IsValid() {
		return errcode.NewValidation("Invalid task priority", nil)
	}
	return nil
}

// AssertWithinProjectTimeline guarantees the zero-bug invariant:
// task.StartDate >= project.StartDate AND task.DueDate <= project.TargetEndDate
func (t *Task) AssertWithinProjectTimeline(project *Project) error {
	taskStart := time.Date(t.StartDate.Year(), t.StartDate.Month(), t.StartDate.Day(), 0, 0, 0, 0, time.UTC)
	taskDue := time.Date(t.DueDate.Year(), t.DueDate.Month(), t.DueDate.Day(), 0, 0, 0, 0, time.UTC)
	projStart := time.Date(project.StartDate.Year(), project.StartDate.Month(), project.StartDate.Day(), 0, 0, 0, 0, time.UTC)
	projEnd := time.Date(project.TargetEndDate.Year(), project.TargetEndDate.Month(), project.TargetEndDate.Day(), 0, 0, 0, 0, time.UTC)

	if taskStart.Before(projStart) {
		return errcode.New(
			errcode.ErrTimelineViolation,
			fmt.Sprintf("Task start date (%s) cannot be earlier than project start date (%s)",
				taskStart.Format("2006-01-02"), projStart.Format("2006-01-02")),
			422,
			map[string]string{"task_start": taskStart.Format("2006-01-02"), "project_start": projStart.Format("2006-01-02")},
		)
	}

	if taskDue.After(projEnd) {
		return errcode.New(
			errcode.ErrTimelineViolation,
			fmt.Sprintf("Task due date (%s) cannot exceed project target end date (%s)",
				taskDue.Format("2006-01-02"), projEnd.Format("2006-01-02")),
			422,
			map[string]string{"task_due": taskDue.Format("2006-01-02"), "project_end": projEnd.Format("2006-01-02")},
		)
	}

	return nil
}

// RemainingHours calculates max(0, EstimatedHours - LoggedHours)
func (t *Task) RemainingHours() float64 {
	if t.Status == TaskStatusDone {
		return 0
	}
	rem := t.EstimatedHours - t.LoggedHours
	if rem < 0 {
		return 0
	}
	return rem
}

// ValidateStatusTransition implements TSK-021: Task Status State Machine & Hand-off Workflow
func ValidateStatusTransition(oldStatus, newStatus TaskStatus, userRole string, hasOpenSubtasks bool) error {
	if oldStatus == newStatus {
		return nil
	}
	if !newStatus.IsValid() {
		return errcode.NewValidation(fmt.Sprintf("Invalid target task status: %s", newStatus), nil)
	}

	// Rule: cannot mark DONE if any subtask is open
	if newStatus == TaskStatusDone && hasOpenSubtasks {
		return errcode.New(
			errcode.ErrValidationFailed,
			"Cannot mark task DONE because one or more child subtasks are still open",
			422,
			nil,
		)
	}

	// Rule: require QA role or PM/Admin to transition from IN_QA to DONE
	if oldStatus == TaskStatusInQA && newStatus == TaskStatusDone {
		roleUpper := strings.ToUpper(userRole)
		isQAOrLead := strings.Contains(roleUpper, "QA") || strings.Contains(roleUpper, "PM") ||
			roleUpper == "PROJECT_MANAGER" || roleUpper == "ADMIN"
		if !isQAOrLead {
			return errcode.New(
				errcode.ErrForbidden,
				"Only QA Engineers, Project Managers, or Admins can verify and mark a task DONE from IN_QA",
				403,
				nil,
			)
		}
	}

	// State machine graph
	validNext := map[TaskStatus][]TaskStatus{
		TaskStatusBacklog:    {TaskStatusTodo, TaskStatusBlocked},
		TaskStatusTodo:       {TaskStatusInProgress, TaskStatusBlocked, TaskStatusBacklog},
		TaskStatusInProgress: {TaskStatusCodeReview, TaskStatusBlocked, TaskStatusTodo},
		TaskStatusCodeReview: {TaskStatusInQA, TaskStatusInProgress, TaskStatusBlocked},
		TaskStatusInQA:       {TaskStatusDone, TaskStatusInProgress, TaskStatusBlocked},
		TaskStatusBlocked:    {TaskStatusBacklog, TaskStatusTodo, TaskStatusInProgress, TaskStatusCodeReview, TaskStatusInQA},
		TaskStatusDone:       {TaskStatusTodo}, // Allowed to reopen
	}

	allowed, exists := validNext[oldStatus]
	if !exists {
		return errcode.NewValidation(fmt.Sprintf("Unknown source task status: %s", oldStatus), nil)
	}

	for _, next := range allowed {
		if next == newStatus {
			return nil
		}
	}

	return errcode.New(
		errcode.ErrValidationFailed,
		fmt.Sprintf("Illegal status transition from %s to %s", oldStatus, newStatus),
		422,
		map[string]string{"from": string(oldStatus), "to": string(newStatus)},
	)
}

// TaskAuditLog stores history of status transitions (TSK-021)
type TaskAuditLog struct {
	ID        string     `json:"id"`
	TaskID    string     `json:"task_id"`
	ChangedBy string     `json:"changed_by"`
	OldStatus TaskStatus `json:"old_status"`
	NewStatus TaskStatus `json:"new_status"`
	Comment   string     `json:"comment"`
	CreatedAt time.Time  `json:"created_at"`
}

// WorkloadDay represents daily required hours for a person
type WorkloadDay struct {
	Date         string  `json:"date"`
	Hours        float64 `json:"hours"`
	CapacityHrs  float64 `json:"capacity_hrs"`
	IsOverloaded bool    `json:"is_overloaded"`
	ExcessHours  float64 `json:"excess_hours"`
}

// AssigneeWorkload models developer daily workload & overload detector (TSK-020)
type AssigneeWorkload struct {
	UserID           string        `json:"user_id"`
	UserName         string        `json:"user_name"`
	Role             string        `json:"role"`
	DailyCapacity    float64       `json:"daily_capacity"`
	TotalAssignedHrs float64       `json:"total_assigned_hrs"`
	IsOverallocated  bool          `json:"is_overallocated"`
	MaxDailyHours    float64       `json:"max_daily_hours"`
	DailyBreakdown   []WorkloadDay `json:"daily_breakdown"`
}

package domain

import "time"

type FeasibilityStatus string

const (
	StatusOnTrack       FeasibilityStatus = "ON_TRACK"
	StatusAtRisk        FeasibilityStatus = "AT_RISK"
	StatusCriticalDelay FeasibilityStatus = "CRITICAL_DELAY"
)

type RoleFeasibilityStatus string

const (
	RoleStatusOnTrack    RoleFeasibilityStatus = "ON_TRACK"
	RoleStatusAtRisk     RoleFeasibilityStatus = "AT_RISK"
	RoleStatusCannotMeet RoleFeasibilityStatus = "CANNOT_MEET"
)

type RoleMetric struct {
	AvailableCapacityHours float64               `json:"available_capacity_hours"`
	RemainingTaskHours     float64               `json:"remaining_task_hours"`
	UtilizationPct         float64               `json:"utilization_pct"`
	Status                 RoleFeasibilityStatus `json:"status"`
}

type MemberBottleneck struct {
	UserID                 string  `json:"user_id"`
	Name                   string  `json:"name"`
	Role                   string  `json:"role"`
	AllocatedTaskHours     float64 `json:"allocated_task_hours"`
	AvailableCapacityHours float64 `json:"available_capacity_hours"`
	IsOverloaded           bool    `json:"is_overloaded"`
}

type FeasibilityReport struct {
	ProjectID               string                      `json:"project_id"`
	ProjectName             string                      `json:"project_name"`
	TargetEndDate           time.Time                   `json:"target_end_date"`
	RemainingWorkingDays    int                         `json:"remaining_working_days"`
	FeasibilityStatus       FeasibilityStatus           `json:"feasibility_status"`
	ProjectedCompletionDate time.Time                   `json:"projected_completion_date"`
	BufferWorkingDays       int                         `json:"buffer_working_days"`
	Roles                   map[string]RoleMetric       `json:"roles"`
	IndividualBottlenecks   []MemberBottleneck          `json:"individual_bottlenecks"`
	CalculatedAt            time.Time                   `json:"calculated_at"`
}

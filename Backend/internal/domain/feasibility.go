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
	Role                   string                `json:"role"`
	MemberCount            int                   `json:"member_count"`
	AvailableCapacityHours float64               `json:"available_capacity_hours"`
	RemainingTaskHours     float64               `json:"remaining_task_hours"`
	BufferOrDeficitHours   float64               `json:"buffer_or_deficit_hours"`
	UtilizationPct         float64               `json:"utilization_pct"`
	Status                 RoleFeasibilityStatus `json:"status"`
}

type MemberBottleneck struct {
	UserID                 string  `json:"user_id"`
	Name                   string  `json:"name"`
	Role                   string  `json:"role"`
	AllocatedTaskHours     float64 `json:"allocated_task_hours"`
	AvailableCapacityHours float64 `json:"available_capacity_hours"`
	DeficitHours           float64 `json:"deficit_hours"`
	IsOverloaded           bool    `json:"is_overloaded"`
	Recommendation         string  `json:"recommendation"`
}

type FeasibilityReport struct {
	ProjectID               string                `json:"project_id"`
	ProjectName             string                `json:"project_name"`
	TargetEndDate           time.Time             `json:"target_end_date"`
	RemainingWorkingDays    int                   `json:"remaining_working_days"`
	CanMeetDeadline         bool                  `json:"can_meet_deadline"`
	FeasibilityScore        float64               `json:"feasibility_score"` // 0 to 100%
	FeasibilityStatus       FeasibilityStatus     `json:"feasibility_status"`
	ProjectedCompletionDate time.Time             `json:"projected_completion_date"`
	DaysAheadOrBehind       int                   `json:"days_ahead_or_behind"`
	TotalRemainingHours     float64               `json:"total_remaining_hours"`
	TotalCapacityHours      float64               `json:"total_capacity_hours"`
	TotalBufferHours        float64               `json:"total_buffer_hours"`
	CriticalPath            []string              `json:"critical_path"` // Task IDs on critical path
	Roles                   map[string]RoleMetric `json:"roles"`
	IndividualBottlenecks   []MemberBottleneck    `json:"individual_bottlenecks"`
	CalculatedAt            time.Time             `json:"calculated_at"`
}

type SimulationRequest struct {
	AdditionalDevs      int     `json:"additional_devs"`
	AdditionalQA        int     `json:"additional_qa"`
	AdditionalBA        int     `json:"additional_ba"`
	OvertimeHoursPerDay float64 `json:"overtime_hours_per_day"`
	ExtendDeadlineDays  int     `json:"extend_deadline_days"`
}

type SimulationResult struct {
	OriginalCanMeetDeadline  bool              `json:"original_can_meet_deadline"`
	SimulatedCanMeetDeadline bool              `json:"simulated_can_meet_deadline"`
	OriginalScore            float64           `json:"original_score"`
	SimulatedScore           float64           `json:"simulated_score"`
	OriginalProjectedDate    time.Time         `json:"original_projected_date"`
	SimulatedProjectedDate   time.Time         `json:"simulated_projected_date"`
	OriginalDaysBuffer       int               `json:"original_days_buffer"`
	SimulatedDaysBuffer      int               `json:"simulated_days_buffer"`
	OriginalReport           FeasibilityReport `json:"original_report"`
	SimulatedReport          FeasibilityReport `json:"simulated_report"`
	RecommendationSummary    string            `json:"recommendation_summary"`
}

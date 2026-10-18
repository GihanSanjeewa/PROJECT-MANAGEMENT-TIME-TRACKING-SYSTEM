package domain

import (
	"math"
	"time"

	"project-management-system/pkg/errcode"
)

type TimeLog struct {
	ID              string     `json:"id"`
	TaskID          string     `json:"task_id"`
	ProjectID       string     `json:"project_id"`
	UserID          string     `json:"user_id"`
	UserName        string     `json:"user_name,omitempty"`
	StartTime       time.Time  `json:"start_time"`
	EndTime         *time.Time `json:"end_time,omitempty"`
	DurationMinutes int        `json:"duration_minutes"`
	Description     string     `json:"description"`
	IsManual        bool       `json:"is_manual"`
	WorkDate        string     `json:"work_date"` // YYYY-MM-DD
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

func (tl *TimeLog) Validate() error {
	if tl.EndTime != nil && tl.EndTime.Before(tl.StartTime) {
		return errcode.NewValidation("Timer end time cannot be earlier than start time", nil)
	}
	if tl.DurationMinutes < 0 {
		return errcode.NewValidation("Duration minutes cannot be negative", nil)
	}
	if tl.DurationMinutes > 1440 {
		return errcode.NewValidation("Duration cannot exceed 24 hours (1440 minutes) in a single log", nil)
	}
	if tl.WorkDate != "" {
		todayUTC := time.Now().UTC().Format("2006-01-02")
		if tl.WorkDate > todayUTC {
			return errcode.NewValidation("Cannot log time for future dates", nil)
		}
	}
	return nil
}

// DurationHours converts integer duration minutes to decimal hours rounded to 2 decimal places
func (tl *TimeLog) DurationHours() float64 {
	if tl.DurationMinutes <= 0 && tl.EndTime != nil {
		diff := tl.EndTime.Sub(tl.StartTime)
		return math.Round(diff.Hours()*100) / 100
	}
	return math.Round((float64(tl.DurationMinutes)/60.0)*100) / 100
}

// TimesheetEntry aggregates time logged on a specific task during a week
type TimesheetEntry struct {
	TaskID       string         `json:"task_id"`
	TaskTitle    string         `json:"task_title"`
	ProjectID    string         `json:"project_id"`
	ProjectName  string         `json:"project_name"`
	DailyMinutes map[string]int `json:"daily_minutes"` // Date (YYYY-MM-DD) -> minutes
	DailyHours   map[string]float64 `json:"daily_hours"` // Date (YYYY-MM-DD) -> hours
	TotalMinutes int            `json:"total_minutes"`
	TotalHours   float64        `json:"total_hours"`
}

// WeeklyTimesheet represents a user's 7-day logged time breakdown
type WeeklyTimesheet struct {
	UserID       string                    `json:"user_id"`
	UserName     string                    `json:"user_name"`
	WeekStart    string                    `json:"week_start"` // YYYY-MM-DD (Monday)
	WeekEnd      string                    `json:"week_end"`   // YYYY-MM-DD (Sunday)
	TotalMinutes int                       `json:"total_minutes"`
	TotalHours   float64                   `json:"total_hours"`
	DailyTotals  map[string]float64        `json:"daily_totals"` // Date -> hours
	Entries      []*TimesheetEntry         `json:"entries"`
}

// BurndownPoint is a single day's snapshot along the project timeline
type BurndownPoint struct {
	Date                 string  `json:"date"` // YYYY-MM-DD
	DayIndex             int     `json:"day_index"`
	IdealRemainingHours  float64 `json:"ideal_remaining_hours"`
	ActualRemainingHours float64 `json:"actual_remaining_hours"`
	LoggedHoursOnDay     float64 `json:"logged_hours_on_day"`
	CumulativeLoggedHours float64 `json:"cumulative_logged_hours"`
}

// ProjectBurndown provides the full ideal vs actual remaining effort projection
type ProjectBurndown struct {
	ProjectID           string           `json:"project_id"`
	ProjectName         string           `json:"project_name"`
	StartDate           string           `json:"start_date"`
	TargetEndDate       string           `json:"target_end_date"`
	TotalEstimatedHours float64          `json:"total_estimated_hours"`
	TotalLoggedHours    float64          `json:"total_logged_hours"`
	Points              []*BurndownPoint `json:"points"`
}

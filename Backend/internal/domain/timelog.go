package domain

import (
	"time"

	"project-management-system/pkg/errcode"
)

type TimeLog struct {
	ID              string     `json:"id"`
	TaskID          string     `json:"task_id"`
	UserID          string     `json:"user_id"`
	StartTime       time.Time  `json:"start_time"`
	EndTime         *time.Time `json:"end_time,omitempty"`
	DurationMinutes int        `json:"duration_minutes"`
	Description     string     `json:"description"`
	IsManual        bool       `json:"is_manual"`
	CreatedAt       time.Time  `json:"created_at"`
}

func (tl *TimeLog) Validate() error {
	if tl.EndTime != nil && tl.EndTime.Before(tl.StartTime) {
		return errcode.NewValidation("Timer end time cannot be earlier than start time", nil)
	}
	if tl.DurationMinutes < 0 {
		return errcode.NewValidation("Duration minutes cannot be negative", nil)
	}
	return nil
}

// DurationHours converts integer duration minutes to decimal hours
func (tl *TimeLog) DurationHours() float64 {
	if tl.DurationMinutes <= 0 && tl.EndTime != nil {
		diff := tl.EndTime.Sub(tl.StartTime)
		return diff.Hours()
	}
	return float64(tl.DurationMinutes) / 60.0
}

package service

import (
	"time"

	"project-management-system/internal/domain"
)

type CalendarService struct{}

func NewCalendarService() *CalendarService {
	return &CalendarService{}
}

// IsWorkingDay checks if a given date is a weekday and not a company holiday
func (c *CalendarService) IsWorkingDay(date time.Time, holidays []*domain.CompanyHoliday) bool {
	weekday := date.Weekday()
	if weekday == time.Saturday || weekday == time.Sunday {
		return false
	}
	dateKey := date.Format("2006-01-02")
	for _, h := range holidays {
		if h.HolidayDate.Format("2006-01-02") == dateKey {
			return false
		}
	}
	return true
}

// CalculateWorkingDays counts net business days in [startDate, endDate]
// strictly excluding Saturdays, Sundays, and matching company holidays.
func (c *CalendarService) CalculateWorkingDays(startDate, endDate time.Time, holidays []*domain.CompanyHoliday) int {
	start := time.Date(startDate.Year(), startDate.Month(), startDate.Day(), 0, 0, 0, 0, time.UTC)
	end := time.Date(endDate.Year(), endDate.Month(), endDate.Day(), 0, 0, 0, 0, time.UTC)

	if end.Before(start) {
		return 0
	}

	holidayMap := make(map[string]bool)
	for _, h := range holidays {
		key := h.HolidayDate.Format("2006-01-02")
		holidayMap[key] = true
	}

	workingDays := 0
	curr := start

	for !curr.After(end) {
		weekday := curr.Weekday()
		// Exclude Saturday and Sunday
		if weekday != time.Saturday && weekday != time.Sunday {
			dateKey := curr.Format("2006-01-02")
			// Exclude company holidays
			if !holidayMap[dateKey] {
				workingDays++
			}
		}
		curr = curr.AddDate(0, 0, 1)
	}

	return workingDays
}

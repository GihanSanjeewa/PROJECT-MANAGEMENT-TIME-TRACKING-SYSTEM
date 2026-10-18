package domain

import (
	"strings"
	"time"

	"project-management-system/pkg/errcode"
)

type CompanyHoliday struct {
	ID          string    `json:"id"`
	HolidayDate time.Time `json:"holiday_date"`
	Name        string    `json:"name"`
	IsRecurring bool      `json:"is_recurring"`
}

func (h *CompanyHoliday) Validate() error {
	if strings.TrimSpace(h.Name) == "" {
		return errcode.NewValidation("Holiday name cannot be blank", nil)
	}
	if h.HolidayDate.IsZero() {
		return errcode.NewValidation("Holiday date must be specified", nil)
	}
	return nil
}

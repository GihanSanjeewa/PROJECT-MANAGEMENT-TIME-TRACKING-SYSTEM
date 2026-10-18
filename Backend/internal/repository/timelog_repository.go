package repository

import (
	"context"
	"sync"
	"time"

	"project-management-system/internal/domain"
	"project-management-system/pkg/errcode"
)

type TimeLogRepository interface {
	Create(ctx context.Context, log *domain.TimeLog) error
	GetByID(ctx context.Context, id string) (*domain.TimeLog, error)
	GetActiveTimer(ctx context.Context, userID string) (*domain.TimeLog, error)
	ListByTask(ctx context.Context, taskID string) ([]*domain.TimeLog, error)
	ListByProject(ctx context.Context, projectID string) ([]*domain.TimeLog, error)
	ListByUser(ctx context.Context, userID string, from, to time.Time) ([]*domain.TimeLog, error)
	Update(ctx context.Context, log *domain.TimeLog) error
	Delete(ctx context.Context, id string) error
	FindOverlapping(ctx context.Context, userID string, excludeID string, start, end time.Time) ([]*domain.TimeLog, error)
	GetTotalLoggedMinutesForUserOnDate(ctx context.Context, userID string, excludeID string, workDate string) (int, error)
	CalculateTaskLoggedMinutes(ctx context.Context, taskID string) (int, error)
}

type InMemoryTimeLogRepository struct {
	mu           sync.RWMutex
	logs         map[string]*domain.TimeLog
	activeTimers map[string]string // key: userID -> logID (mirrors PostgreSQL partial unique index WHERE end_time IS NULL)
}

func NewInMemoryTimeLogRepository() *InMemoryTimeLogRepository {
	return &InMemoryTimeLogRepository{
		logs:         make(map[string]*domain.TimeLog),
		activeTimers: make(map[string]string),
	}
}

func (r *InMemoryTimeLogRepository) Create(ctx context.Context, log *domain.TimeLog) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Zero-Bug Concurrency Gate: Exactly 1 active timer per user (TSK-026)
	if log.EndTime == nil {
		if existingID, exists := r.activeTimers[log.UserID]; exists {
			// Verify the referenced log is actually still active in storage
			if activeLog, ok := r.logs[existingID]; ok && activeLog.EndTime == nil {
				return errcode.NewTimerAlreadyActive("An active timer is already running for this user")
			}
			delete(r.activeTimers, log.UserID)
		}
		r.activeTimers[log.UserID] = log.ID
	}

	now := time.Now().UTC()
	if log.CreatedAt.IsZero() {
		log.CreatedAt = now
	}
	log.UpdatedAt = now

	cp := *log
	r.logs[log.ID] = &cp
	return nil
}

func (r *InMemoryTimeLogRepository) GetByID(ctx context.Context, id string) (*domain.TimeLog, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	l, exists := r.logs[id]
	if !exists {
		return nil, errcode.NewNotFound("Time log not found")
	}
	cp := *l
	return &cp, nil
}

func (r *InMemoryTimeLogRepository) GetActiveTimer(ctx context.Context, userID string) (*domain.TimeLog, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	logID, exists := r.activeTimers[userID]
	if !exists {
		return nil, nil
	}

	l, ok := r.logs[logID]
	if !ok || l.EndTime != nil {
		return nil, nil
	}

	cp := *l
	return &cp, nil
}

func (r *InMemoryTimeLogRepository) ListByTask(ctx context.Context, taskID string) ([]*domain.TimeLog, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	res := make([]*domain.TimeLog, 0)
	for _, l := range r.logs {
		if l.TaskID == taskID {
			cp := *l
			res = append(res, &cp)
		}
	}
	return res, nil
}

func (r *InMemoryTimeLogRepository) ListByProject(ctx context.Context, projectID string) ([]*domain.TimeLog, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	res := make([]*domain.TimeLog, 0)
	for _, l := range r.logs {
		if l.ProjectID == projectID {
			cp := *l
			res = append(res, &cp)
		}
	}
	return res, nil
}

func (r *InMemoryTimeLogRepository) ListByUser(ctx context.Context, userID string, from, to time.Time) ([]*domain.TimeLog, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	res := make([]*domain.TimeLog, 0)
	for _, l := range r.logs {
		if l.UserID == userID {
			if !from.IsZero() && l.StartTime.Before(from) {
				continue
			}
			if !to.IsZero() && l.StartTime.After(to) {
				continue
			}
			cp := *l
			res = append(res, &cp)
		}
	}
	return res, nil
}

func (r *InMemoryTimeLogRepository) Update(ctx context.Context, log *domain.TimeLog) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	existing, exists := r.logs[log.ID]
	if !exists {
		return errcode.NewNotFound("Time log not found")
	}

	// If timer was stopped, remove from activeTimers mapping
	if existing.EndTime == nil && log.EndTime != nil {
		delete(r.activeTimers, log.UserID)
	} else if existing.EndTime != nil && log.EndTime == nil {
		// Attempting to reopen as active timer
		if activeID, activeExists := r.activeTimers[log.UserID]; activeExists && activeID != log.ID {
			return errcode.NewTimerAlreadyActive("Another active timer is already running for this user")
		}
		r.activeTimers[log.UserID] = log.ID
	}

	log.UpdatedAt = time.Now().UTC()
	cp := *log
	r.logs[log.ID] = &cp
	return nil
}

func (r *InMemoryTimeLogRepository) Delete(ctx context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	existing, exists := r.logs[id]
	if !exists {
		return errcode.NewNotFound("Time log not found")
	}

	if existing.EndTime == nil {
		delete(r.activeTimers, existing.UserID)
	}

	delete(r.logs, id)
	return nil
}

func (r *InMemoryTimeLogRepository) FindOverlapping(ctx context.Context, userID string, excludeID string, start, end time.Time) ([]*domain.TimeLog, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	res := make([]*domain.TimeLog, 0)
	for _, l := range r.logs {
		if l.UserID != userID || l.ID == excludeID || l.EndTime == nil {
			continue
		}

		// Interval intersection: start < other.End && end > other.Start
		if start.Before(*l.EndTime) && end.After(l.StartTime) {
			cp := *l
			res = append(res, &cp)
		}
	}
	return res, nil
}

func (r *InMemoryTimeLogRepository) GetTotalLoggedMinutesForUserOnDate(ctx context.Context, userID string, excludeID string, workDate string) (int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	total := 0
	for _, l := range r.logs {
		if l.UserID == userID && l.ID != excludeID && l.WorkDate == workDate {
			total += l.DurationMinutes
		}
	}
	return total, nil
}

func (r *InMemoryTimeLogRepository) CalculateTaskLoggedMinutes(ctx context.Context, taskID string) (int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	total := 0
	for _, l := range r.logs {
		if l.TaskID == taskID {
			total += l.DurationMinutes
		}
	}
	return total, nil
}

package repository

import (
	"context"
	"fmt"
	"sync"
	"time"

	"project-management-system/internal/domain"
	"project-management-system/pkg/errcode"
)

type ProjectRepository interface {
	Create(ctx context.Context, project *domain.Project) error
	GetByID(ctx context.Context, id string) (*domain.Project, error)
	GetByCode(ctx context.Context, code string) (*domain.Project, error)
	List(ctx context.Context) ([]*domain.Project, error)
	Update(ctx context.Context, project *domain.Project) error

	// Member management
	AddMember(ctx context.Context, member *domain.ProjectMember) error
	GetMembers(ctx context.Context, projectID string) ([]*domain.ProjectMember, error)
	RemoveMember(ctx context.Context, projectID, userID string) error
	GetMember(ctx context.Context, projectID, userID string) (*domain.ProjectMember, error)

	// Company holidays
	GetHolidays(ctx context.Context) ([]*domain.CompanyHoliday, error)
	AddHoliday(ctx context.Context, holiday *domain.CompanyHoliday) error

	// Task bounds check for timeline shrinking guard
	GetTaskDateBounds(ctx context.Context, projectID string) (*time.Time, *time.Time, []string, error)
}

type InMemoryProjectRepository struct {
	mu       sync.RWMutex
	projects map[string]*domain.Project
	members  map[string]*domain.ProjectMember // key: projectID + ":" + userID + ":" + role
	holidays []*domain.CompanyHoliday
	// Tasks store reference for timeline guard check
	tasks map[string]*domain.Task
}

func NewInMemoryProjectRepository() *InMemoryProjectRepository {
	repo := &InMemoryProjectRepository{
		projects: make(map[string]*domain.Project),
		members:  make(map[string]*domain.ProjectMember),
		holidays: make([]*domain.CompanyHoliday, 0),
		tasks:    make(map[string]*domain.Task),
	}

	// Seed holidays (e.g. Christmas, New Year)
	repo.holidays = append(repo.holidays,
		&domain.CompanyHoliday{
			ID:          "hol-1",
			HolidayDate: time.Date(2026, 12, 25, 0, 0, 0, 0, time.UTC),
			Name:        "Christmas Day",
			IsRecurring: true,
		},
		&domain.CompanyHoliday{
			ID:          "hol-2",
			HolidayDate: time.Date(2026, 11, 26, 0, 0, 0, 0, time.UTC),
			Name:        "Thanksgiving Day",
			IsRecurring: false,
		},
	)

	return repo
}

// SetTaskForGuard registers a task in the repo (used by Task service or tests)
func (r *InMemoryProjectRepository) SetTaskForGuard(task *domain.Task) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.tasks[task.ID] = task
}

func (r *InMemoryProjectRepository) Create(ctx context.Context, project *domain.Project) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, p := range r.projects {
		if p.Code == project.Code {
			return errcode.NewConflict(fmt.Sprintf("Project with code '%s' already exists", project.Code), map[string]string{"code": project.Code})
		}
	}

	r.projects[project.ID] = project
	return nil
}

func (r *InMemoryProjectRepository) GetByID(ctx context.Context, id string) (*domain.Project, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	p, ok := r.projects[id]
	if !ok {
		return nil, errcode.NewNotFound("Project not found")
	}
	copy := *p
	return &copy, nil
}

func (r *InMemoryProjectRepository) GetByCode(ctx context.Context, code string) (*domain.Project, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, p := range r.projects {
		if p.Code == code {
			copy := *p
			return &copy, nil
		}
	}
	return nil, errcode.NewNotFound("Project not found")
}

func (r *InMemoryProjectRepository) List(ctx context.Context) ([]*domain.Project, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	res := make([]*domain.Project, 0, len(r.projects))
	for _, p := range r.projects {
		copy := *p
		res = append(res, &copy)
	}
	return res, nil
}

func (r *InMemoryProjectRepository) Update(ctx context.Context, project *domain.Project) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	existing, ok := r.projects[project.ID]
	if !ok {
		return errcode.NewNotFound("Project not found")
	}

	for id, p := range r.projects {
		if id != project.ID && p.Code == project.Code {
			return errcode.NewConflict("Project code already exists", map[string]string{"code": project.Code})
		}
	}

	project.CreatedAt = existing.CreatedAt
	project.UpdatedAt = time.Now().UTC()
	r.projects[project.ID] = project
	return nil
}

func (r *InMemoryProjectRepository) AddMember(ctx context.Context, member *domain.ProjectMember) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	key := fmt.Sprintf("%s:%s:%s", member.ProjectID, member.UserID, member.ProjectRole)
	if _, exists := r.members[key]; exists {
		return errcode.NewConflict("User is already assigned to this project with this role", map[string]string{
			"user_id": member.UserID,
			"role":    string(member.ProjectRole),
		})
	}

	r.members[key] = member
	return nil
}

func (r *InMemoryProjectRepository) GetMembers(ctx context.Context, projectID string) ([]*domain.ProjectMember, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	res := make([]*domain.ProjectMember, 0)
	for _, m := range r.members {
		if m.ProjectID == projectID {
			copy := *m
			res = append(res, &copy)
		}
	}
	return res, nil
}

func (r *InMemoryProjectRepository) RemoveMember(ctx context.Context, projectID, userID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	deleted := false
	for k, m := range r.members {
		if m.ProjectID == projectID && m.UserID == userID {
			delete(r.members, k)
			deleted = true
		}
	}

	if !deleted {
		return errcode.NewNotFound("Project member not found")
	}
	return nil
}

func (r *InMemoryProjectRepository) GetMember(ctx context.Context, projectID, userID string) (*domain.ProjectMember, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, m := range r.members {
		if m.ProjectID == projectID && m.UserID == userID {
			copy := *m
			return &copy, nil
		}
	}
	return nil, errcode.NewNotFound("Project member not found")
}

func (r *InMemoryProjectRepository) GetHolidays(ctx context.Context) ([]*domain.CompanyHoliday, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	res := make([]*domain.CompanyHoliday, len(r.holidays))
	copy(res, r.holidays)
	return res, nil
}

func (r *InMemoryProjectRepository) AddHoliday(ctx context.Context, holiday *domain.CompanyHoliday) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.holidays = append(r.holidays, holiday)
	return nil
}

// GetTaskDateBounds inspects all tasks in the project to verify timeline bounds
func (r *InMemoryProjectRepository) GetTaskDateBounds(ctx context.Context, projectID string) (*time.Time, *time.Time, []string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var minStart *time.Time
	var maxDue *time.Time
	taskIDs := make([]string, 0)

	for _, t := range r.tasks {
		if t.ProjectID == projectID {
			taskIDs = append(taskIDs, t.ID)
			if minStart == nil || t.StartDate.Before(*minStart) {
				s := t.StartDate
				minStart = &s
			}
			if maxDue == nil || t.DueDate.After(*maxDue) {
				d := t.DueDate
				maxDue = &d
			}
		}
	}

	return minStart, maxDue, taskIDs, nil
}

func (r *InMemoryProjectRepository) SyncTask(t *domain.Task) {
	r.mu.Lock()
	defer r.mu.Unlock()
	copy := *t
	r.tasks[t.ID] = &copy
}

func (r *InMemoryProjectRepository) RemoveTask(id string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.tasks, id)
}

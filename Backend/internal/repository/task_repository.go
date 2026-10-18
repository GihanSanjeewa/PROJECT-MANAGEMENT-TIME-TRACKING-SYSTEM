package repository

import (
	"context"
	"sync"
	"time"

	"project-management-system/internal/domain"
	"project-management-system/pkg/errcode"
)

type TaskRepository interface {
	Create(ctx context.Context, task *domain.Task) error
	GetByID(ctx context.Context, id string) (*domain.Task, error)
	ListByProject(ctx context.Context, projectID string) ([]*domain.Task, error)
	Update(ctx context.Context, task *domain.Task) error
	Delete(ctx context.Context, id string) error
	GetSubtasks(ctx context.Context, parentTaskID string) ([]*domain.Task, error)

	// Dependencies
	AddDependency(ctx context.Context, dep *domain.TaskDependency) error
	GetDependencies(ctx context.Context, projectID string) ([]*domain.TaskDependency, error)
	GetDependenciesForTask(ctx context.Context, taskID string) ([]*domain.TaskDependency, error)
	DeleteDependency(ctx context.Context, id string) error

	// Audit Logs
	AddAuditLog(ctx context.Context, log *domain.TaskAuditLog) error
	GetAuditLogs(ctx context.Context, taskID string) ([]*domain.TaskAuditLog, error)
}

type InMemoryTaskRepository struct {
	mu           sync.RWMutex
	tasks        map[string]*domain.Task
	dependencies map[string]*domain.TaskDependency // key: id
	auditLogs    []*domain.TaskAuditLog
	projectRepo  *InMemoryProjectRepository
}

func NewInMemoryTaskRepository(projectRepo *InMemoryProjectRepository) *InMemoryTaskRepository {
	return &InMemoryTaskRepository{
		tasks:        make(map[string]*domain.Task),
		dependencies: make(map[string]*domain.TaskDependency),
		auditLogs:    make([]*domain.TaskAuditLog, 0),
		projectRepo:  projectRepo,
	}
}

func (r *InMemoryTaskRepository) Create(ctx context.Context, task *domain.Task) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now().UTC()
	task.CreatedAt = now
	task.UpdatedAt = now

	copy := *task
	r.tasks[task.ID] = &copy

	if r.projectRepo != nil {
		r.projectRepo.SyncTask(task)
	}
	return nil
}

func (r *InMemoryTaskRepository) GetByID(ctx context.Context, id string) (*domain.Task, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	t, exists := r.tasks[id]
	if !exists {
		return nil, errcode.NewNotFound("Task not found")
	}
	copy := *t
	return &copy, nil
}

func (r *InMemoryTaskRepository) ListByProject(ctx context.Context, projectID string) ([]*domain.Task, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	res := make([]*domain.Task, 0)
	for _, t := range r.tasks {
		if t.ProjectID == projectID {
			copy := *t
			res = append(res, &copy)
		}
	}
	return res, nil
}

func (r *InMemoryTaskRepository) Update(ctx context.Context, task *domain.Task) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.tasks[task.ID]; !exists {
		return errcode.NewNotFound("Task not found")
	}

	task.UpdatedAt = time.Now().UTC()
	copy := *task
	r.tasks[task.ID] = &copy

	if r.projectRepo != nil {
		r.projectRepo.SyncTask(task)
	}
	return nil
}

func (r *InMemoryTaskRepository) Delete(ctx context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.tasks[id]; !exists {
		return errcode.NewNotFound("Task not found")
	}

	delete(r.tasks, id)

	// Clean up associated dependencies
	for depID, dep := range r.dependencies {
		if dep.TaskID == id || dep.DependsOnTaskID == id {
			delete(r.dependencies, depID)
		}
	}

	if r.projectRepo != nil {
		r.projectRepo.RemoveTask(id)
	}
	return nil
}

func (r *InMemoryTaskRepository) GetSubtasks(ctx context.Context, parentTaskID string) ([]*domain.Task, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	res := make([]*domain.Task, 0)
	for _, t := range r.tasks {
		if t.ParentTaskID != nil && *t.ParentTaskID == parentTaskID {
			copy := *t
			res = append(res, &copy)
		}
	}
	return res, nil
}

func (r *InMemoryTaskRepository) AddDependency(ctx context.Context, dep *domain.TaskDependency) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Check unique constraint
	for _, existing := range r.dependencies {
		if existing.TaskID == dep.TaskID && existing.DependsOnTaskID == dep.DependsOnTaskID {
			return errcode.NewConflict("Dependency already exists", nil)
		}
	}

	copy := *dep
	r.dependencies[dep.ID] = &copy
	return nil
}

func (r *InMemoryTaskRepository) GetDependencies(ctx context.Context, projectID string) ([]*domain.TaskDependency, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	res := make([]*domain.TaskDependency, 0)
	for _, dep := range r.dependencies {
		if dep.ProjectID == projectID {
			copy := *dep
			res = append(res, &copy)
		}
	}
	return res, nil
}

func (r *InMemoryTaskRepository) GetDependenciesForTask(ctx context.Context, taskID string) ([]*domain.TaskDependency, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	res := make([]*domain.TaskDependency, 0)
	for _, dep := range r.dependencies {
		if dep.TaskID == taskID {
			copy := *dep
			res = append(res, &copy)
		}
	}
	return res, nil
}

func (r *InMemoryTaskRepository) DeleteDependency(ctx context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.dependencies[id]; !exists {
		return errcode.NewNotFound("Dependency not found")
	}
	delete(r.dependencies, id)
	return nil
}

func (r *InMemoryTaskRepository) AddAuditLog(ctx context.Context, log *domain.TaskAuditLog) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if log.CreatedAt.IsZero() {
		log.CreatedAt = time.Now().UTC()
	}
	copy := *log
	r.auditLogs = append(r.auditLogs, &copy)
	return nil
}

func (r *InMemoryTaskRepository) GetAuditLogs(ctx context.Context, taskID string) ([]*domain.TaskAuditLog, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	res := make([]*domain.TaskAuditLog, 0)
	for _, l := range r.auditLogs {
		if l.TaskID == taskID {
			copy := *l
			res = append(res, &copy)
		}
	}
	return res, nil
}

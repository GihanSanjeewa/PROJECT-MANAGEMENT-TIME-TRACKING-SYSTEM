-- 000003_tasks_and_timelogs.up.sql
CREATE TABLE IF NOT EXISTS tasks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    parent_task_id UUID REFERENCES tasks(id) ON DELETE CASCADE,
    title VARCHAR(255) NOT NULL,
    description TEXT,
    assigned_to UUID REFERENCES users(id) ON DELETE SET NULL,
    required_role VARCHAR(50) NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'TODO',
    priority VARCHAR(50) NOT NULL DEFAULT 'MEDIUM',
    estimated_hours NUMERIC(8,2) NOT NULL,
    logged_hours NUMERIC(8,2) NOT NULL DEFAULT 0.00,
    start_date DATE NOT NULL,
    due_date DATE NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT chk_task_dates CHECK (due_date >= start_date),
    CONSTRAINT chk_task_estimated CHECK (estimated_hours > 0),
    CONSTRAINT chk_task_logged CHECK (logged_hours >= 0)
);

CREATE INDEX IF NOT EXISTS idx_tasks_project ON tasks(project_id);
CREATE INDEX IF NOT EXISTS idx_tasks_assignee ON tasks(assigned_to);
CREATE INDEX IF NOT EXISTS idx_tasks_status ON tasks(status);
CREATE INDEX IF NOT EXISTS idx_tasks_dates ON tasks(start_date, due_date);

CREATE TABLE IF NOT EXISTS task_dependencies (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    task_id UUID NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    depends_on_task_id UUID NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    dependency_type VARCHAR(20) NOT NULL DEFAULT 'FS',
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT chk_not_self_dependency CHECK (task_id <> depends_on_task_id),
    CONSTRAINT uq_task_dependency UNIQUE (task_id, depends_on_task_id)
);

CREATE TABLE IF NOT EXISTS time_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    task_id UUID NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    start_time TIMESTAMPTZ NOT NULL,
    end_time TIMESTAMPTZ NULL,
    duration_minutes INT NOT NULL DEFAULT 0,
    description TEXT,
    is_manual BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT chk_timer_end CHECK (end_time IS NULL OR end_time >= start_time),
    CONSTRAINT chk_duration_positive CHECK (duration_minutes >= 0)
);

-- Mandatory Zero-Bug Concurrency Gate: Exactly 1 active timer per user
CREATE UNIQUE INDEX IF NOT EXISTS idx_one_active_timer ON time_logs(user_id) WHERE end_time IS NULL;
CREATE INDEX IF NOT EXISTS idx_time_logs_user_start ON time_logs(user_id, start_time);
CREATE INDEX IF NOT EXISTS idx_time_logs_task ON time_logs(task_id);

-- 000002_projects_and_members.up.sql
CREATE TABLE IF NOT EXISTS projects (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code VARCHAR(50) NOT NULL UNIQUE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    start_date DATE NOT NULL,
    target_end_date DATE NOT NULL,
    budget_hours NUMERIC(10,2) NOT NULL DEFAULT 0.00,
    status VARCHAR(50) NOT NULL DEFAULT 'DRAFT',
    created_by UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT chk_projects_dates CHECK (target_end_date >= start_date),
    CONSTRAINT chk_projects_budget CHECK (budget_hours >= 0)
);

CREATE INDEX IF NOT EXISTS idx_projects_status ON projects(status);
CREATE INDEX IF NOT EXISTS idx_projects_dates ON projects(start_date, target_end_date);

CREATE TABLE IF NOT EXISTS project_members (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    project_role VARCHAR(50) NOT NULL,
    allocation_pct INT NOT NULL DEFAULT 100,
    effective_daily_hours NUMERIC(4,2) NOT NULL DEFAULT 8.00,
    joined_date DATE NOT NULL,
    left_date DATE NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT chk_member_allocation CHECK (allocation_pct BETWEEN 1 AND 100),
    CONSTRAINT chk_member_daily_hours CHECK (effective_daily_hours > 0 AND effective_daily_hours <= 24),
    CONSTRAINT chk_member_left_date CHECK (left_date IS NULL OR left_date >= joined_date),
    CONSTRAINT uq_project_member_role UNIQUE (project_id, user_id, project_role)
);

CREATE INDEX IF NOT EXISTS idx_project_members_lookup ON project_members(project_id, user_id);

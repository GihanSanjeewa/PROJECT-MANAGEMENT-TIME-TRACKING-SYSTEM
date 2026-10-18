-- 000003_tasks_and_timelogs.down.sql
DROP INDEX IF EXISTS idx_one_active_timer;
DROP TABLE IF EXISTS time_logs;
DROP TABLE IF EXISTS task_dependencies;
DROP TABLE IF EXISTS tasks;

-- 009_background_tasks.down.sql

DROP INDEX IF EXISTS idx_task_steps_order;
DROP INDEX IF EXISTS idx_agent_tasks_user;
DROP INDEX IF EXISTS idx_agent_tasks_status;
DROP TABLE IF EXISTS task_steps;
DROP TABLE IF EXISTS agent_tasks;

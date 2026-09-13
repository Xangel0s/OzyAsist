-- backend/internal/db/migrations/012_security_and_audit.down.sql

DROP TABLE IF EXISTS shadow_snapshots;
DROP TABLE IF EXISTS audit_trail_chained;

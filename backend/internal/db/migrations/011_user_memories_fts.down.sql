-- backend/internal/db/migrations/011_user_memories_fts.down.sql

DROP TRIGGER IF EXISTS trg_user_memories_au;
DROP TRIGGER IF EXISTS trg_user_memories_ad;
DROP TRIGGER IF EXISTS trg_user_memories_ai;
DROP TABLE IF EXISTS user_memories_fts;
DROP TABLE IF EXISTS user_memories;

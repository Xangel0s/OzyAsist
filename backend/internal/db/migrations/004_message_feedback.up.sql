ALTER TABLE messages ADD COLUMN feedback TEXT DEFAULT '' CHECK(feedback IN ('', 'like', 'dislike'));

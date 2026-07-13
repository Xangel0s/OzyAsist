ALTER TABLE projects ADD COLUMN agent_consent TEXT DEFAULT 'ask' CHECK(agent_consent IN ('ask', 'always', 'never'));

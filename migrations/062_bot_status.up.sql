CREATE TABLE IF NOT EXISTS bot_status_config (
    id INTEGER PRIMARY KEY DEFAULT 1,
    activity_type INTEGER NOT NULL,
    name TEXT NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Ensure only one row can exist by enforcing id=1
ALTER TABLE bot_status_config ADD CONSTRAINT single_row_bot_status CHECK (id = 1);

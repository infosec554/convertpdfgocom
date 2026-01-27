-- Add premium_until column to bot_users table
ALTER TABLE bot_users ADD COLUMN IF NOT EXISTS premium_until TIMESTAMP;

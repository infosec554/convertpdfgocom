-- Remove premium_until column from bot_users table
ALTER TABLE bot_users DROP COLUMN IF EXISTS premium_until;

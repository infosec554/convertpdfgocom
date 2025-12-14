-- WARNING: Casting back to UUID will fail if any non-UUID values were inserted.
DELETE FROM files WHERE user_id !~ '^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$';
ALTER TABLE files ALTER COLUMN user_id TYPE UUID USING user_id::UUID;
ALTER TABLE files ADD CONSTRAINT files_user_id_fkey FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;

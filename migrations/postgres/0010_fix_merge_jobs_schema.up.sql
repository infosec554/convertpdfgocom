-- Fix missing input_file_ids column in merge_jobs if it doesn't exist
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='merge_jobs' AND column_name='input_file_ids') THEN
        ALTER TABLE merge_jobs ADD COLUMN input_file_ids TEXT[];
    END IF;
END $$;

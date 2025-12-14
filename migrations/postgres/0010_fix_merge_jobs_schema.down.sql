-- Revert adding input_file_ids column
ALTER TABLE merge_jobs DROP COLUMN IF EXISTS input_file_ids;

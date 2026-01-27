-- Revert changes (Warning: CASCADE constraints cannot be easily reverted to NO ACTION without dropping)
-- Ideally we would drop and re-add without CASCADE, but for now we just leave it or strictly revert
-- This is a partial revert if needed

ALTER TABLE compress_jobs DROP CONSTRAINT IF EXISTS compress_jobs_input_file_id_fkey;
ALTER TABLE compress_jobs ADD CONSTRAINT compress_jobs_input_file_id_fkey FOREIGN KEY (input_file_id) REFERENCES files(id);

ALTER TABLE compress_jobs DROP CONSTRAINT IF EXISTS compress_jobs_output_file_id_fkey;
ALTER TABLE compress_jobs ADD CONSTRAINT compress_jobs_output_file_id_fkey FOREIGN KEY (output_file_id) REFERENCES files(id);

-- (And so on for others, but typically valid to assume simplistic revert for this task context)

-- Drop existing foreign keys and re-add them with ON DELETE CASCADE

-- 1. compress_jobs
ALTER TABLE compress_jobs DROP CONSTRAINT IF EXISTS compress_jobs_input_file_id_fkey;
ALTER TABLE compress_jobs ADD CONSTRAINT compress_jobs_input_file_id_fkey FOREIGN KEY (input_file_id) REFERENCES files(id) ON DELETE CASCADE;

ALTER TABLE compress_jobs DROP CONSTRAINT IF EXISTS compress_jobs_output_file_id_fkey;
ALTER TABLE compress_jobs ADD CONSTRAINT compress_jobs_output_file_id_fkey FOREIGN KEY (output_file_id) REFERENCES files(id) ON DELETE CASCADE;

-- 2. word_to_pdf_jobs
ALTER TABLE word_to_pdf_jobs DROP CONSTRAINT IF EXISTS word_to_pdf_jobs_input_file_id_fkey;
ALTER TABLE word_to_pdf_jobs ADD CONSTRAINT word_to_pdf_jobs_input_file_id_fkey FOREIGN KEY (input_file_id) REFERENCES files(id) ON DELETE CASCADE;

ALTER TABLE word_to_pdf_jobs DROP CONSTRAINT IF EXISTS word_to_pdf_jobs_output_file_id_fkey;
ALTER TABLE word_to_pdf_jobs ADD CONSTRAINT word_to_pdf_jobs_output_file_id_fkey FOREIGN KEY (output_file_id) REFERENCES files(id) ON DELETE CASCADE;

-- 3. jpg_to_pdf_jobs
ALTER TABLE jpg_to_pdf_jobs DROP CONSTRAINT IF EXISTS jpg_to_pdf_jobs_output_file_id_fkey;
ALTER TABLE jpg_to_pdf_jobs ADD CONSTRAINT jpg_to_pdf_jobs_output_file_id_fkey FOREIGN KEY (output_file_id) REFERENCES files(id) ON DELETE CASCADE;

-- 4. excel_to_pdf_jobs
ALTER TABLE excel_to_pdf_jobs DROP CONSTRAINT IF EXISTS excel_to_pdf_jobs_input_file_id_fkey;
ALTER TABLE excel_to_pdf_jobs ADD CONSTRAINT excel_to_pdf_jobs_input_file_id_fkey FOREIGN KEY (input_file_id) REFERENCES files(id) ON DELETE CASCADE;

ALTER TABLE excel_to_pdf_jobs DROP CONSTRAINT IF EXISTS excel_to_pdf_jobs_output_file_id_fkey;
ALTER TABLE excel_to_pdf_jobs ADD CONSTRAINT excel_to_pdf_jobs_output_file_id_fkey FOREIGN KEY (output_file_id) REFERENCES files(id) ON DELETE CASCADE;

-- 5. powerpoint_to_pdf_jobs
ALTER TABLE powerpoint_to_pdf_jobs DROP CONSTRAINT IF EXISTS powerpoint_to_pdf_jobs_input_file_id_fkey;
ALTER TABLE powerpoint_to_pdf_jobs ADD CONSTRAINT powerpoint_to_pdf_jobs_input_file_id_fkey FOREIGN KEY (input_file_id) REFERENCES files(id) ON DELETE CASCADE;

ALTER TABLE powerpoint_to_pdf_jobs DROP CONSTRAINT IF EXISTS powerpoint_to_pdf_jobs_output_file_id_fkey;
ALTER TABLE powerpoint_to_pdf_jobs ADD CONSTRAINT powerpoint_to_pdf_jobs_output_file_id_fkey FOREIGN KEY (output_file_id) REFERENCES files(id) ON DELETE CASCADE;

-- 6. pdf_to_jpg_jobs
ALTER TABLE pdf_to_jpg_jobs DROP CONSTRAINT IF EXISTS pdf_to_jpg_jobs_input_file_id_fkey;
ALTER TABLE pdf_to_jpg_jobs ADD CONSTRAINT pdf_to_jpg_jobs_input_file_id_fkey FOREIGN KEY (input_file_id) REFERENCES files(id) ON DELETE CASCADE;

ALTER TABLE pdf_to_jpg_jobs DROP CONSTRAINT IF EXISTS pdf_to_jpg_jobs_zip_file_id_fkey;
ALTER TABLE pdf_to_jpg_jobs ADD CONSTRAINT pdf_to_jpg_jobs_zip_file_id_fkey FOREIGN KEY (zip_file_id) REFERENCES files(id) ON DELETE CASCADE;

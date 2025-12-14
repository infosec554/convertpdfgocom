-- Add ON DELETE CASCADE to remaining job tables

-- 1. split_jobs
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.table_constraints WHERE constraint_name='split_jobs_input_file_id_fkey') THEN
        ALTER TABLE split_jobs DROP CONSTRAINT split_jobs_input_file_id_fkey;
        ALTER TABLE split_jobs ADD CONSTRAINT split_jobs_input_file_id_fkey FOREIGN KEY (input_file_id) REFERENCES files(id) ON DELETE CASCADE;
    END IF;
    
    IF EXISTS (SELECT 1 FROM information_schema.table_constraints WHERE constraint_name='split_jobs_output_file_id_fkey') THEN
        ALTER TABLE split_jobs DROP CONSTRAINT split_jobs_output_file_id_fkey;
        ALTER TABLE split_jobs ADD CONSTRAINT split_jobs_output_file_id_fkey FOREIGN KEY (output_file_id) REFERENCES files(id) ON DELETE CASCADE;
    END IF;
END $$;

-- 2. rotate_jobs
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.table_constraints WHERE constraint_name='rotate_jobs_input_file_id_fkey') THEN
        ALTER TABLE rotate_jobs DROP CONSTRAINT rotate_jobs_input_file_id_fkey;
        ALTER TABLE rotate_jobs ADD CONSTRAINT rotate_jobs_input_file_id_fkey FOREIGN KEY (input_file_id) REFERENCES files(id) ON DELETE CASCADE;
    END IF;

    IF EXISTS (SELECT 1 FROM information_schema.table_constraints WHERE constraint_name='rotate_jobs_output_file_id_fkey') THEN
        ALTER TABLE rotate_jobs DROP CONSTRAINT rotate_jobs_output_file_id_fkey;
        ALTER TABLE rotate_jobs ADD CONSTRAINT rotate_jobs_output_file_id_fkey FOREIGN KEY (output_file_id) REFERENCES files(id) ON DELETE CASCADE;
    END IF;
END $$;

-- 3. merge_jobs (Check output_file_id only, input_file_ids is array)
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.table_constraints WHERE constraint_name='merge_jobs_output_file_id_fkey') THEN
        ALTER TABLE merge_jobs DROP CONSTRAINT merge_jobs_output_file_id_fkey;
        ALTER TABLE merge_jobs ADD CONSTRAINT merge_jobs_output_file_id_fkey FOREIGN KEY (output_file_id) REFERENCES files(id) ON DELETE CASCADE;
    END IF;
END $$;

-- 4. watermark_jobs
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.table_constraints WHERE constraint_name='watermark_jobs_input_file_id_fkey') THEN
        ALTER TABLE watermark_jobs DROP CONSTRAINT watermark_jobs_input_file_id_fkey;
        ALTER TABLE watermark_jobs ADD CONSTRAINT watermark_jobs_input_file_id_fkey FOREIGN KEY (input_file_id) REFERENCES files(id) ON DELETE CASCADE;
    END IF;

    IF EXISTS (SELECT 1 FROM information_schema.table_constraints WHERE constraint_name='watermark_jobs_output_file_id_fkey') THEN
        ALTER TABLE watermark_jobs DROP CONSTRAINT watermark_jobs_output_file_id_fkey;
        ALTER TABLE watermark_jobs ADD CONSTRAINT watermark_jobs_output_file_id_fkey FOREIGN KEY (output_file_id) REFERENCES files(id) ON DELETE CASCADE;
    END IF;
END $$;

-- 5. unlock_jobs
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.table_constraints WHERE constraint_name='unlock_jobs_input_file_id_fkey') THEN
        ALTER TABLE unlock_jobs DROP CONSTRAINT unlock_jobs_input_file_id_fkey;
        ALTER TABLE unlock_jobs ADD CONSTRAINT unlock_jobs_input_file_id_fkey FOREIGN KEY (input_file_id) REFERENCES files(id) ON DELETE CASCADE;
    END IF;

    IF EXISTS (SELECT 1 FROM information_schema.table_constraints WHERE constraint_name='unlock_jobs_output_file_id_fkey') THEN
        ALTER TABLE unlock_jobs DROP CONSTRAINT unlock_jobs_output_file_id_fkey;
        ALTER TABLE unlock_jobs ADD CONSTRAINT unlock_jobs_output_file_id_fkey FOREIGN KEY (output_file_id) REFERENCES files(id) ON DELETE CASCADE;
    END IF;
END $$;

-- 6. security_jobs (Protect PDF)
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.table_constraints WHERE constraint_name='security_jobs_input_file_id_fkey') THEN
        ALTER TABLE security_jobs DROP CONSTRAINT security_jobs_input_file_id_fkey;
        ALTER TABLE security_jobs ADD CONSTRAINT security_jobs_input_file_id_fkey FOREIGN KEY (input_file_id) REFERENCES files(id) ON DELETE CASCADE;
    END IF;

    IF EXISTS (SELECT 1 FROM information_schema.table_constraints WHERE constraint_name='security_jobs_output_file_id_fkey') THEN
        ALTER TABLE security_jobs DROP CONSTRAINT security_jobs_output_file_id_fkey;
        ALTER TABLE security_jobs ADD CONSTRAINT security_jobs_output_file_id_fkey FOREIGN KEY (output_file_id) REFERENCES files(id) ON DELETE CASCADE;
    END IF;
END $$;

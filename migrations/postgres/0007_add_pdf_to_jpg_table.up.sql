CREATE TABLE IF NOT EXISTS pdf_to_jpg_jobs (
    id UUID PRIMARY KEY,
    user_id VARCHAR(64),
    input_file_id UUID NOT NULL REFERENCES files(id),
    output_file_ids TEXT[], -- Array of file IDs if multiple (optional, better zip)
    zip_file_id UUID REFERENCES files(id),
    status VARCHAR(20) DEFAULT 'created',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

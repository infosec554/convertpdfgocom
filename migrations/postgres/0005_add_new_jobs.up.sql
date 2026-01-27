CREATE TABLE IF NOT EXISTS merge_jobs (
    id UUID PRIMARY KEY,
    user_id BIGINT REFERENCES bot_users(telegram_id),
    input_file_ids TEXT[], -- Array of file IDs
    output_file_id TEXT,
    status VARCHAR(50) DEFAULT 'pending',
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS split_jobs (
    id UUID PRIMARY KEY,
    user_id BIGINT REFERENCES bot_users(telegram_id),
    input_file_id TEXT,
    page_range VARCHAR(50),
    output_file_id TEXT,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS security_jobs (
    id UUID PRIMARY KEY,
    user_id BIGINT REFERENCES bot_users(telegram_id),
    input_file_id TEXT,
    type VARCHAR(20), -- 'protect' or 'unlock'
    password VARCHAR(255),
    output_file_id TEXT,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS pdf_to_image_jobs (
    id UUID PRIMARY KEY,
    user_id BIGINT REFERENCES bot_users(telegram_id),
    input_file_id TEXT,
    output_file_id TEXT, -- Likely a zip file ID
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS web_to_pdf_jobs (
    id UUID PRIMARY KEY,
    user_id BIGINT REFERENCES bot_users(telegram_id),
    url TEXT,
    output_file_id TEXT,
    created_at TIMESTAMP DEFAULT NOW()
);

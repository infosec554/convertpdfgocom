DO $$ BEGIN
    CREATE TYPE user_status_enum AS ENUM ('active', 'deleted', 'blocked');
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

DO $$ BEGIN
    CREATE TYPE user_role_enum AS ENUM ('user', 'admin');
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    status user_status_enum NOT NULL DEFAULT 'active',
    role user_role_enum NOT NULL DEFAULT 'user', -- "admin" yoki "user"
    name VARCHAR(255) NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    password VARCHAR(255) NOT NULL,
    created_at TIMESTAMP DEFAULT NOW()
   updated_at TIMESTAMP DEFAULT NOW()

);

ALTER TABLE users 
ADD COLUMN language VARCHAR(50) DEFAULT 'en',   -- Foydalanuvchi tilini saqlash
ADD COLUMN notifications BOOLEAN DEFAULT TRUE;   -- Bildirishnomalarni olishni xohlayaptimi

CREATE TABLE user_preferences (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,  -- Foydalanuvchi ID
    language VARCHAR(50) DEFAULT 'en',  -- Foydalanuvchi tilini saqlash (default: 'en')
    notifications BOOLEAN DEFAULT TRUE,  -- Foydalanuvchi bildirishnomalarni olishni xohlayaptimi
    created_at TIMESTAMP DEFAULT NOW(), 
    updated_at TIMESTAMP DEFAULT NOW()
);



CREATE TABLE password_reset_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    token VARCHAR(255) UNIQUE NOT NULL,  -- Parolni tiklash uchun token
    created_at TIMESTAMP DEFAULT NOW(), 
    expires_at TIMESTAMP                -- Tokenning amal qilish muddati
);


CREATE TABLE files (
    id UUID PRIMARY KEY,
    user_id UUID REFERENCES users(id) ON DELETE CASCADE, 
    file_name VARCHAR(255) NOT NULL,
    file_path TEXT NOT NULL,
    file_type VARCHAR(255) NOT NULL,
    file_size INTEGER NOT NULL,
    uploaded_at TIMESTAMP DEFAULT NOW()
);

-- ORGANIZE PDF
CREATE TABLE organize_jobs (
    id UUID PRIMARY KEY,
    user_id UUID REFERENCES users(id),
    input_file_id UUID NOT NULL REFERENCES files(id),
    new_order INTEGER[] NOT NULL, -- Sahifalar yangi tartibi
    output_file_id UUID REFERENCES files(id),
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    created_at TIMESTAMP DEFAULT NOW()
);

-- MERGE PDF
CREATE TABLE merge_jobs (
    id UUID PRIMARY KEY,
    user_id UUID REFERENCES users(id),
    output_file_id UUID REFERENCES files(id),
    status VARCHAR(20) NOT NULL,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE merge_job_input_files (
    id UUID PRIMARY KEY,
    job_id UUID NOT NULL REFERENCES merge_jobs(id) ON DELETE CASCADE,
    file_id UUID NOT NULL REFERENCES files(id)
);

CREATE TABLE split_jobs (
    id UUID PRIMARY KEY,
    user_id UUID REFERENCES users(id),
    input_file_id UUID NOT NULL REFERENCES files(id),
    split_ranges TEXT NOT NULL,
    output_file_ids UUID[] DEFAULT ARRAY[]::UUID[],
    status VARCHAR(20) NOT NULL,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE remove_pages_jobs (
    id UUID PRIMARY KEY,
    user_id UUID REFERENCES users(id),
    input_file_id UUID NOT NULL REFERENCES files(id),
    pages_to_remove TEXT NOT NULL,
    output_file_id UUID REFERENCES files(id),
    status VARCHAR(20) NOT NULL,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE extract_pages_jobs (
    id UUID PRIMARY KEY,
    user_id UUID REFERENCES users(id),
    input_file_id UUID NOT NULL REFERENCES files(id),
    pages_to_extract TEXT NOT NULL,
    output_file_id UUID REFERENCES files(id),
    status VARCHAR(20) NOT NULL,
    created_at TIMESTAMP DEFAULT NOW()
);

-- OPTIMIZE PDF
CREATE TABLE compress_jobs (
    id UUID PRIMARY KEY,
    user_id UUID REFERENCES users(id),
    input_file_id UUID NOT NULL REFERENCES files(id),
    compression VARCHAR(10),
    output_file_id UUID REFERENCES files(id),
    status VARCHAR(20) NOT NULL,
    created_at TIMESTAMP DEFAULT NOW()
);

-- CONVERT TO PDF
CREATE TABLE jpg_to_pdf_jobs (
    id UUID PRIMARY KEY,
    user_id UUID REFERENCES users(id),
    input_file_ids UUID[] NOT NULL, -- Bir nechta rasm fayllari
    output_file_id UUID REFERENCES files(id),
    status VARCHAR(20) NOT NULL CHECK (status IN ('pending', 'processing', 'done', 'failed')),
    created_at TIMESTAMP DEFAULT NOW()
);



CREATE TABLE pdf_to_jpg_jobs (
    id UUID PRIMARY KEY,
    user_id UUID REFERENCES users(id),
    input_file_id UUID NOT NULL REFERENCES files(id),
    output_file_ids UUID[] DEFAULT ARRAY[]::UUID[],
    zip_file_id UUID REFERENCES files(id),
    status VARCHAR(20) NOT NULL CHECK (status IN ('pending', 'processing', 'done', 'failed')),
    created_at TIMESTAMP DEFAULT NOW()
);


CREATE TABLE rotate_jobs (
    id UUID PRIMARY KEY,
    user_id UUID REFERENCES users(id),
    input_file_id UUID NOT NULL REFERENCES files(id),
    rotation_angle INTEGER NOT NULL,   -- Burilish burchagi
    pages VARCHAR NOT NULL DEFAULT 'all', -- Sahifa diapazoni
    output_file_id UUID REFERENCES files(id),  -- Chiqish fayli
    output_path VARCHAR,  -- Faylni saqlash joyi
    status VARCHAR(20) NOT NULL,  -- Holat: 'pending', 'done'
    created_at TIMESTAMP DEFAULT now() -- Yaratilgan vaqti
);


CREATE TABLE add_page_number_jobs (
  id UUID PRIMARY KEY,
  user_id UUID REFERENCES users(id),
  input_file_id UUID NOT NULL REFERENCES files(id),
  output_file_id UUID REFERENCES files(id),
  status VARCHAR(20) NOT NULL,
  first_number INT,
  page_range VARCHAR(50),
  position VARCHAR(20),
  color VARCHAR(20),
  font_size INT,
  created_at TIMESTAMP DEFAULT now()
);


CREATE TABLE add_watermark_jobs (
    id UUID PRIMARY KEY,
     user_id UUID REFERENCES users(id),
    input_file_id UUID NOT NULL,
    output_file_id UUID,
    text TEXT NOT NULL,
    font_name TEXT NOT NULL,
    font_size INTEGER NOT NULL,
    position TEXT NOT NULL,
    rotation INTEGER DEFAULT 0,
    opacity DOUBLE PRECISION DEFAULT 1.0,
    fill_color TEXT NOT NULL,
    pages TEXT DEFAULT 'all',
    status VARCHAR(20) NOT NULL,
    created_at TIMESTAMP NOT NULL
);




CREATE TABLE crop_pdf_jobs (
    id UUID PRIMARY KEY,
    user_id UUID REFERENCES users(id),
    input_file_id UUID NOT NULL REFERENCES files(id),
    top INTEGER NOT NULL,
    bottom INTEGER NOT NULL,
    "left" INTEGER NOT NULL,
    "right" INTEGER NOT NULL,
    output_file_id UUID REFERENCES files(id),
    status VARCHAR(20) NOT NULL,
    created_at TIMESTAMP DEFAULT NOW()
);

-- PDF SECURITY
CREATE TABLE unlock_jobs (
    id UUID PRIMARY KEY,
    user_id UUID REFERENCES users(id),
    input_file_id UUID NOT NULL REFERENCES files(id),
    output_file_id UUID REFERENCES files(id),
    status VARCHAR(20) NOT NULL,
    created_at TIMESTAMP DEFAULT NOW()
);

-- Logs Table
CREATE TABLE logs (
    id UUID PRIMARY KEY,
    job_id UUID,
    job_type VARCHAR(30),
    message TEXT,
    level VARCHAR(10),
    created_at TIMESTAMP DEFAULT NOW()
);



-- Shared Links
CREATE TABLE shared_links (
    id UUID PRIMARY KEY,
    file_id UUID REFERENCES files(id) ON DELETE CASCADE,
    shared_token VARCHAR(255) UNIQUE NOT NULL,
    expires_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT NOW()
);
CREATE TABLE protect_jobs (
    id UUID PRIMARY KEY,
    user_id UUID REFERENCES users(id),
    input_file_id UUID NOT NULL REFERENCES files(id),
    output_file_id UUID REFERENCES files(id),
    password TEXT NOT NULL,
    status VARCHAR(20) NOT NULL,
    created_at TIMESTAMP DEFAULT NOW()
);
CREATE TABLE pdf_to_word_jobs (
    id UUID PRIMARY KEY,
    user_id UUID REFERENCES users(id),
    input_file_id UUID NOT NULL REFERENCES files(id),
    output_file_id UUID REFERENCES files(id),
    status VARCHAR(50) NOT NULL CHECK (
        status IN ('pending', 'processing', 'done', 'failed', 'conversion_failed')
    ),
    created_at TIMESTAMP DEFAULT NOW()
);
CREATE TABLE word_to_pdf_jobs (
    id UUID PRIMARY KEY,
    user_id UUID REFERENCES users(id),
    input_file_id UUID NOT NULL REFERENCES files(id),
    output_file_id UUID REFERENCES files(id),
    status VARCHAR(50) NOT NULL CHECK (status IN ('pending', 'processing', 'done', 'failed')),
    created_at TIMESTAMP DEFAULT NOW()
);
CREATE TABLE excel_to_pdf_jobs (
    id UUID PRIMARY KEY,
    user_id UUID REFERENCES users(id),
    input_file_id UUID NOT NULL REFERENCES files(id),
    output_file_id UUID REFERENCES files(id),
    status VARCHAR(20) NOT NULL CHECK (status IN ('pending', 'processing', 'done', 'failed')),
    created_at TIMESTAMP DEFAULT now()
);

CREATE TABLE powerpoint_to_pdf_jobs (
    id UUID PRIMARY KEY,
    user_id UUID REFERENCES users(id),
    input_file_id UUID NOT NULL REFERENCES files(id),
    output_file_id UUID REFERENCES files(id),
    status VARCHAR(20) NOT NULL CHECK (status IN ('pending', 'processing', 'done', 'failed')),
    created_at TIMESTAMP DEFAULT now()
);

CREATE TABLE files_deletion_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    file_id UUID NOT NULL,
    user_id UUID NULL,           -- fayl egasi, agar ma’lum bo‘lsa
    deleted_by UUID NOT NULL,    -- o‘chirgan foydalanuvchi yoki sistema
    deleted_at TIMESTAMP NOT NULL DEFAULT NOW(),
    reason TEXT NULL             -- o‘chirish sababi (ixtiyoriy)
);
CREATE TABLE contact_messages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    email VARCHAR(255) NOT NULL,
    message TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT NOW()
);
CREATE TABLE  contact_messages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        VARCHAR(255) NOT NULL,
    email       VARCHAR(255) NOT NULL,
    subject     VARCHAR(200) NOT NULL,
    message     TEXT         NOT NULL,
    terms_accepted BOOLEAN   NOT NULL DEFAULT false,
    created_at  TIMESTAMP    NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_contact_messages_created_at
ON contact_messages(created_at DESC);

ALTER TABLE contact_messages
    ADD COLUMN IF NOT EXISTS is_read   BOOLEAN   NOT NULL DEFAULT false,
    ADD COLUMN IF NOT EXISTS replied_at TIMESTAMP NULL;

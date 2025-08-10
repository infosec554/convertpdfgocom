package postgres

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"

	"convertpdfgo/api/models"
	"convertpdfgo/pkg/logger"
	"convertpdfgo/storage"
)

type adminJobRepo struct {
	db  *pgxpool.Pool
	log logger.ILogger
}

func NewAdminJobRepo(db *pgxpool.Pool, log logger.ILogger) storage.IAdminJobStorage {
	return &adminJobRepo{db: db, log: log}
}

func (r *adminJobRepo) ListJobs(ctx context.Context, f models.AdminJobFilter) ([]models.JobSummary, error) {
	// 🔰 Barcha joblar uchun yagona view
	base := `
		-- ORGANIZE
		SELECT id, 'organize'::text AS job_type, status, user_id, output_file_id::uuid, created_at, NULL::timestamp AS updated_at
		FROM organize_jobs

		UNION ALL
		-- MERGE
		SELECT id, 'merge', status, user_id, output_file_id::uuid, created_at, NULL::timestamp
		FROM merge_jobs

		UNION ALL
		-- SPLIT (ko‘p output -> NULL)
		SELECT id, 'split', status, user_id, NULL::uuid AS output_file_id, created_at, NULL::timestamp
		FROM split_jobs

		UNION ALL
		-- REMOVE PAGES
		SELECT id, 'remove_pages', status, user_id, output_file_id::uuid, created_at, NULL::timestamp
		FROM remove_pages_jobs

		UNION ALL
		-- EXTRACT PAGES
		SELECT id, 'extract_pages', status, user_id, output_file_id::uuid, created_at, NULL::timestamp
		FROM extract_pages_jobs

		UNION ALL
		-- COMPRESS
		SELECT id, 'compress', status, user_id, output_file_id::uuid, created_at, NULL::timestamp
		FROM compress_jobs

		UNION ALL
		-- JPG TO PDF
		SELECT id, 'jpg_to_pdf', status, user_id, output_file_id::uuid, created_at, NULL::timestamp
		FROM jpg_to_pdf_jobs

		UNION ALL
		-- PDF TO JPG (zip_file_id -> output)
		SELECT id, 'pdf_to_jpg', status, user_id, zip_file_id::uuid AS output_file_id, created_at, NULL::timestamp
		FROM pdf_to_jpg_jobs

		UNION ALL
		-- ROTATE
		SELECT id, 'rotate', status, user_id, output_file_id::uuid, created_at, NULL::timestamp
		FROM rotate_jobs

		UNION ALL
		-- ADD PAGE NUMBER
		SELECT id, 'add_page_number', status, user_id, output_file_id::uuid, created_at, NULL::timestamp
		FROM add_page_number_jobs

		UNION ALL
		-- ADD WATERMARK
		SELECT id, 'add_watermark', status, user_id, output_file_id::uuid, created_at, NULL::timestamp
		FROM add_watermark_jobs

		UNION ALL
		-- CROP PDF
		SELECT id, 'crop_pdf', status, user_id, output_file_id::uuid, created_at, NULL::timestamp
		FROM crop_pdf_jobs

		UNION ALL
		-- UNLOCK
		SELECT id, 'unlock', status, user_id, output_file_id::uuid, created_at, NULL::timestamp
		FROM unlock_jobs

		UNION ALL
		-- PROTECT
		SELECT id, 'protect', status, user_id, output_file_id::uuid, created_at, NULL::timestamp
		FROM protect_jobs

		UNION ALL
		-- PDF TO WORD
		SELECT id, 'pdf_to_word', status, user_id, output_file_id::uuid, created_at, NULL::timestamp
		FROM pdf_to_word_jobs

		UNION ALL
		-- WORD TO PDF
		SELECT id, 'word_to_pdf', status, user_id, output_file_id::uuid, created_at, NULL::timestamp
		FROM word_to_pdf_jobs

		UNION ALL
		-- EXCEL TO PDF
		SELECT id, 'excel_to_pdf', status, user_id, output_file_id::uuid, created_at, NULL::timestamp
		FROM excel_to_pdf_jobs

		UNION ALL
		-- POWERPOINT TO PDF
		SELECT id, 'powerpoint_to_pdf', status, user_id, output_file_id::uuid, created_at, NULL::timestamp
		FROM powerpoint_to_pdf_jobs
	`

	var sb strings.Builder
	sb.WriteString(`
		SELECT id, job_type, status, user_id, output_file_id, created_at, updated_at
		FROM (`)
	sb.WriteString(base)
	sb.WriteString(`) jobs WHERE 1=1`)

	args := []any{}
	argn := 1
	add := func(cond string, v any) {
		sb.WriteString(" AND ")
		sb.WriteString(cond)
		args = append(args, v)
		argn++
	}

	if f.Type != nil && *f.Type != "" {
		add(fmt.Sprintf("job_type = $%d", argn), *f.Type)
	}
	if f.Status != nil && *f.Status != "" {
		add(fmt.Sprintf("status = $%d", argn), *f.Status)
	}
	if f.UserID != nil && *f.UserID != "" {
		add(fmt.Sprintf("user_id = $%d", argn), *f.UserID)
	}
	if f.From != nil {
		add(fmt.Sprintf("created_at >= $%d", argn), *f.From)
	}
	if f.To != nil {
		add(fmt.Sprintf("created_at <= $%d", argn), *f.To)
	}
	if f.Search != nil && *f.Search != "" {
		add(fmt.Sprintf("id ILIKE $%d", argn), *f.Search+"%")
	}

	if f.Limit <= 0 || f.Limit > 200 {
		f.Limit = 50
	}
	if f.Offset < 0 {
		f.Offset = 0
	}
	sb.WriteString(" ORDER BY created_at DESC")
	sb.WriteString(fmt.Sprintf(" LIMIT $%d OFFSET $%d", argn, argn+1))
	args = append(args, f.Limit, f.Offset)

	query := sb.String()

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		r.log.Error("admin ListJobs query error", logger.Error(err))
		return nil, err
	}
	defer rows.Close()

	var out []models.JobSummary
	for rows.Next() {
		var js models.JobSummary
		if err := rows.Scan(&js.ID, &js.JobType, &js.Status, &js.UserID, &js.OutputFileID, &js.CreatedAt, &js.UpdatedAt); err != nil {
			r.log.Error("scan job summary error", logger.Error(err))
			continue
		}
		out = append(out, js)
	}
	if rows.Err() != nil {
		return nil, rows.Err()
	}
	return out, nil
}

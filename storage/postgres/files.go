package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"

	"convertpdfgo/api/models"
	"convertpdfgo/pkg/logger"
	"convertpdfgo/storage"
)

type fileRepo struct {
	db  *pgxpool.Pool
	log logger.ILogger
}

func NewFileRepo(db *pgxpool.Pool, log logger.ILogger) storage.IFileStorage {
	return &fileRepo{
		db:  db,
		log: log,
	}
}

// Save - faylni DBga yozish
func (f *fileRepo) Save(ctx context.Context, file models.File) (string, error) {
	query := `
		INSERT INTO files (id, user_id, file_name, file_path, file_type, file_size, uploaded_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	_, err := f.db.Exec(ctx, query,
		file.ID, file.UserID, file.FileName, file.FilePath,
		file.FileType, file.FileSize, file.UploadedAt)
	if err != nil {
		f.log.Error("DB insert error", logger.Error(err))
		return "", err
	}
	return file.ID, nil
}

// GetByID - faylni ID bo‘yicha olish
func (f *fileRepo) GetByID(ctx context.Context, id string) (models.File, error) {
	var file models.File
	var userID sql.NullString

	query := `
        SELECT id, user_id, file_name, file_path, file_type, file_size, uploaded_at
        FROM files WHERE id = $1
    `
	err := f.db.QueryRow(ctx, query, id).Scan(
		&file.ID, &userID, &file.FileName, &file.FilePath,
		&file.FileType, &file.FileSize, &file.UploadedAt,
	)
	if err != nil {
		f.log.Error("failed to fetch file", logger.Error(err))
		return models.File{}, err
	}

	if userID.Valid {
		v := userID.String
		file.UserID = &v
	} else {
		file.UserID = nil
	}
	return file, nil
}

// Delete - faylni ID bo‘yicha o‘chirish
func (f *fileRepo) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM files WHERE id = $1`
	_, err := f.db.Exec(ctx, query, id)
	if err != nil {
		f.log.Error("failed to delete file", logger.Error(err))
	}
	return err
}

// ListByUser - foydalanuvchiga tegishli fayllar

func (f *fileRepo) ListByUser(ctx context.Context, userID string) ([]models.File, error) {
	query := `
		SELECT id, user_id, file_name, file_path, file_type, file_size, uploaded_at
		FROM files WHERE user_id = $1 ORDER BY uploaded_at DESC
	`
	rows, err := f.db.Query(ctx, query, userID)
	if err != nil {
		f.log.Error("failed to fetch user files", logger.Error(err))
		return nil, err
	}
	defer rows.Close()

	var files []models.File
	for rows.Next() {
		var file models.File
		var uid sql.NullString

		if err := rows.Scan(
			&file.ID,
			&uid,
			&file.FileName,
			&file.FilePath,
			&file.FileType,
			&file.FileSize,
			&file.UploadedAt,
		); err != nil {
			f.log.Error("error scanning row", logger.Error(err))
			continue
		}
		if uid.Valid {
			v := uid.String
			file.UserID = &v
		} else {
			file.UserID = nil
		}
		files = append(files, file)
	}
	return files, nil
}

func (r *fileRepo) GetOldFiles(ctx context.Context, olderThanDays int) ([]models.OldFile, error) {
	query := `
		SELECT id, file_path
		FROM files
		WHERE uploaded_at < NOW() - ($1 * INTERVAL '1 day')
	`

	rows, err := r.db.Query(ctx, query, olderThanDays)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var oldFiles []models.OldFile
	for rows.Next() {
		var f models.OldFile
		if err := rows.Scan(&f.ID, &f.FilePath); err != nil {
			continue
		}
		oldFiles = append(oldFiles, f)
	}
	return oldFiles, rows.Err()
}

func (r *fileRepo) DeleteByID(ctx context.Context, id string) error {
	query := `DELETE FROM files WHERE id = $1`
	_, err := r.db.Exec(ctx, query, id)
	return err
}

func (r *fileRepo) GetPendingDeletionFiles(ctx context.Context, expirationMinutes int) ([]models.File, error) {
	query := `
		SELECT id, user_id, file_name, file_path, file_type, file_size, uploaded_at
		FROM files
		WHERE uploaded_at < NOW() - ($1 * INTERVAL '1 minute')
		ORDER BY uploaded_at ASC
	`

	rows, err := r.db.Query(ctx, query, expirationMinutes)
	if err != nil {
		r.log.Error("failed to fetch pending deletion files", logger.Error(err))
		return nil, err
	}
	defer rows.Close()

	var files []models.File
	for rows.Next() {
		var file models.File
		if err := rows.Scan(
			&file.ID,
			&file.UserID, // *string bo‘lsa NULL ham to‘g‘ri tushadi
			&file.FileName,
			&file.FilePath,
			&file.FileType,
			&file.FileSize,
			&file.UploadedAt,
		); err != nil {
			r.log.Error("error scanning pending deletion file row", logger.Error(err))
			continue
		}
		files = append(files, file)
	}

	if rows.Err() != nil {
		r.log.Error("error iterating pending deletion file rows", logger.Error(rows.Err()))
		return nil, rows.Err()
	}

	return files, nil
}

func (r *fileRepo) AdminListFiles(ctx context.Context, f models.AdminFileFilter) ([]models.FileRow, error) {
	var (
		where  []string
		args   []interface{}
		argPos = 1
	)

	// UserID filtri
	if f.UserID != nil && *f.UserID != "" {
		where = append(where, fmt.Sprintf("files.user_id = $%d", argPos))
		args = append(args, *f.UserID)
		argPos++
	} else if f.IncludeGuests {
		// faqat guestlar yoki hammasi? bu yerda "hammasi" (NULL ham mumkin)
		// includeGuests true bo'lsa — hech narsa qo'shmaymiz (hammasi chiqadi)
	} else {
		// guestlarni chiqarma (faqat userga tegishlilar)
		where = append(where, "files.user_id IS NOT NULL")
	}

	// Qidiruv (file_name LIKE)
	if f.Q != nil && *f.Q != "" {
		where = append(where, fmt.Sprintf("files.file_name ILIKE $%d", argPos))
		args = append(args, "%"+*f.Q+"%")
		argPos++
	}

	// Sana oraliqlari
	if f.DateFrom != nil {
		where = append(where, fmt.Sprintf("files.uploaded_at >= $%d", argPos))
		args = append(args, *f.DateFrom)
		argPos++
	}
	if f.DateTo != nil {
		where = append(where, fmt.Sprintf("files.uploaded_at <= $%d", argPos))
		args = append(args, *f.DateTo)
		argPos++
	}

	if f.Limit <= 0 {
		f.Limit = 20
	}
	if f.Offset < 0 {
		f.Offset = 0
	}

	sb := strings.Builder{}
	sb.WriteString(`
		SELECT
			files.id,
			files.user_id,
			files.file_name,
			files.file_path,
			files.file_type,
			files.file_size,
			files.uploaded_at,
			u.email
		FROM files
		LEFT JOIN users u ON u.id = files.user_id
	`)
	if len(where) > 0 {
		sb.WriteString(" WHERE ")
		sb.WriteString(strings.Join(where, " AND "))
	}
	sb.WriteString(" ORDER BY files.uploaded_at DESC ")
	sb.WriteString(fmt.Sprintf(" LIMIT %d OFFSET %d", f.Limit, f.Offset))

	rows, err := r.db.Query(ctx, sb.String(), args...)
	if err != nil {
		r.log.Error("admin list files query error", logger.Error(err))
		return nil, err
	}
	defer rows.Close()

	var out []models.FileRow
	for rows.Next() {
		var row models.FileRow
		var uid sql.NullString
		var email sql.NullString

		if err := rows.Scan(
			&row.ID, &uid, &row.FileName, &row.FilePath, &row.FileType, &row.FileSize, &row.UploadedAt, &email,
		); err != nil {
			r.log.Error("scan error", logger.Error(err))
			continue
		}
		if uid.Valid {
			v := uid.String
			row.UserID = &v
		}
		if email.Valid {
			v := email.String
			row.UserEmail = &v
		}
		out = append(out, row)
	}
	if rows.Err() != nil {
		return nil, rows.Err()
	}
	return out, nil
}

func (r *fileRepo) AdminCountFiles(ctx context.Context, f models.AdminFileFilter) (int64, error) {
	var (
		where  []string
		args   []interface{}
		argPos = 1
	)

	if f.UserID != nil && *f.UserID != "" {
		where = append(where, fmt.Sprintf("files.user_id = $%d", argPos))
		args = append(args, *f.UserID)
		argPos++
	} else if f.IncludeGuests {
		// all
	} else {
		where = append(where, "files.user_id IS NOT NULL")
	}

	if f.Q != nil && *f.Q != "" {
		where = append(where, fmt.Sprintf("files.file_name ILIKE $%d", argPos))
		args = append(args, "%"+*f.Q+"%")
		argPos++
	}
	if f.DateFrom != nil {
		where = append(where, fmt.Sprintf("files.uploaded_at >= $%d", argPos))
		args = append(args, *f.DateFrom)
		argPos++
	}
	if f.DateTo != nil {
		where = append(where, fmt.Sprintf("files.uploaded_at <= $%d", argPos))
		args = append(args, *f.DateTo)
		argPos++
	}

	sb := strings.Builder{}
	sb.WriteString(`SELECT COUNT(*) FROM files`)
	if len(where) > 0 {
		sb.WriteString(" WHERE ")
		sb.WriteString(strings.Join(where, " AND "))
	}

	var cnt int64
	if err := r.db.QueryRow(ctx, sb.String(), args...).Scan(&cnt); err != nil {
		return 0, err
	}
	return cnt, nil
}

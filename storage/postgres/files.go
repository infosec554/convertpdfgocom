package postgres

import (
	"context"
	"database/sql"

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
	}
	return file, nil
}

func (f *fileRepo) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM files WHERE id = $1`
	_, err := f.db.Exec(ctx, query, id)
	if err != nil {
		f.log.Error("failed to delete file", logger.Error(err))
	}
	return err
}

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
			&file.ID, &uid, &file.FileName, &file.FilePath,
			&file.FileType, &file.FileSize, &file.UploadedAt,
		); err != nil {
			f.log.Error("error scanning row", logger.Error(err))
			continue
		}
		if uid.Valid {
			v := uid.String
			file.UserID = &v
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
			&file.ID, &file.UserID, &file.FileName, &file.FilePath,
			&file.FileType, &file.FileSize, &file.UploadedAt,
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

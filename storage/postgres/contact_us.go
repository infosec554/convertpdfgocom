package postgres

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"convertpdfgo/api/models"
	"convertpdfgo/pkg/logger"
	"convertpdfgo/storage"
)

type contactRepo struct {
	db  *pgxpool.Pool
	log logger.ILogger
}

func NewContactRepo(db *pgxpool.Pool, log logger.ILogger) storage.IContactStorage {
	return &contactRepo{db: db, log: log}
}

func (r *contactRepo) Create(ctx context.Context, req models.ContactCreateRequest, id string) error {
	q := `INSERT INTO contact_messages (id, name, email, message, created_at, is_read, subject)
          VALUES ($1,$2,$3,$4,NOW(),false,$5)`
	_, err := r.db.Exec(ctx, q, id, req.Name, req.Email, req.Message, req.Subject)
	return err
}

func (r *contactRepo) GetByID(ctx context.Context, id string) (*models.ContactMessage, error) {
	q := `SELECT id,name,email,subject,message,is_read,created_at,replied_at
          FROM contact_messages WHERE id=$1`
	var m models.ContactMessage
	err := r.db.QueryRow(ctx, q, id).Scan(
		&m.ID, &m.Name, &m.Email, &m.Subject, &m.Message, &m.IsRead, &m.CreatedAt, &m.RepliedAt,
	)
	if err != nil {
		return nil, err
	}
	return &m, nil
}

func (r *contactRepo) List(ctx context.Context, onlyUnread bool, limit, offset int) ([]models.ContactMessage, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	base := `SELECT id,name,email,subject,message,is_read,created_at,replied_at
             FROM contact_messages`
	if onlyUnread {
		base += ` WHERE is_read=false`
	}
	base += ` ORDER BY created_at DESC LIMIT $1 OFFSET $2`
	rows, err := r.db.Query(ctx, base, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.ContactMessage
	for rows.Next() {
		var m models.ContactMessage
		if err := rows.Scan(&m.ID, &m.Name, &m.Email, &m.Subject, &m.Message, &m.IsRead, &m.CreatedAt, &m.RepliedAt); err == nil {
			out = append(out, m)
		}
	}
	return out, rows.Err()
}

func (r *contactRepo) MarkRead(ctx context.Context, id string) error {
	_, err := r.db.Exec(ctx, `UPDATE contact_messages SET is_read=true WHERE id=$1`, id)
	return err
}

func (r *contactRepo) Delete(ctx context.Context, id string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM contact_messages WHERE id=$1`, id)
	return err
}
func (r *contactRepo) SaveReply(ctx context.Context, id, adminID, subject, body string, repliedAt time.Time) error {
	q := `
		UPDATE contact_messages
		   SET replied_at = $1,
		       replied_by = $2,
		       reply_subject = $3,
		       reply_body = $4
		 WHERE id = $5
	`
	_, err := r.db.Exec(ctx, q, repliedAt, adminID, subject, body, id)
	return err
}

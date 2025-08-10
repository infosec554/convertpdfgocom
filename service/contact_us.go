package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"convertpdfgo/api/models"
	"convertpdfgo/pkg/logger"
	"convertpdfgo/storage"
)

type ContactService interface {
	// Public
	Create(ctx context.Context, req models.ContactCreateRequest) (string, error)

	// Admin
	List(ctx context.Context, onlyUnread bool, limit, offset int) ([]models.ContactMessage, error)
	GetByID(ctx context.Context, id string) (*models.ContactMessage, error)
	MarkRead(ctx context.Context, id string) error
	Delete(ctx context.Context, id string) error
}

type contactService struct {
	stg storage.IStorage
	log logger.ILogger
}

func NewContactService(stg storage.IStorage, log logger.ILogger) ContactService {
	return &contactService{stg: stg, log: log}
}

func (s *contactService) Create(ctx context.Context, req models.ContactCreateRequest) (string, error) {
	// Biznes qoidasi: Terms qabul qilingan bo‘lsin
	if !req.TermsAccepted {
		return "", ErrTermsNotAccepted
	}
	id := uuid.NewString()

	s.log.Info("contact received", logger.String("email", req.Email))
	if err := s.stg.Contact().Create(ctx, req, id); err != nil {
		s.log.Error("failed to save contact", logger.Error(err))
		return "", err
	}
	_ = time.Now() // DB default created_at bor, faqat log uchun kerak bo‘lsa ishlatamiz

	return id, nil
}

func (s *contactService) List(ctx context.Context, onlyUnread bool, limit, offset int) ([]models.ContactMessage, error) {
	msgs, err := s.stg.Contact().List(ctx, onlyUnread, limit, offset)
	if err != nil {
		s.log.Error("failed to list contacts", logger.Error(err))
		return nil, err
	}
	return msgs, nil
}

func (s *contactService) GetByID(ctx context.Context, id string) (*models.ContactMessage, error) {
	msg, err := s.stg.Contact().GetByID(ctx, id)
	if err != nil {
		s.log.Error("failed to get contact by id", logger.Error(err))
		return nil, err
	}
	return msg, nil
}

func (s *contactService) MarkRead(ctx context.Context, id string) error {
	if err := s.stg.Contact().MarkRead(ctx, id); err != nil {
		s.log.Error("failed to mark contact read", logger.Error(err))
		return err
	}
	return nil
}

func (s *contactService) Delete(ctx context.Context, id string) error {
	if err := s.stg.Contact().Delete(ctx, id); err != nil {
		s.log.Error("failed to delete contact", logger.Error(err))
		return err
	}
	return nil
}

// Global xato
var ErrTermsNotAccepted = fmt.Errorf("terms must be accepted")

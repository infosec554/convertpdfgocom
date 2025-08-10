package service

import "convertpdfgo/pkg/mailer"

// MailerService interfeysi
type MailerService interface {
	Send(to, subject, body string) error
}

// mailerService structi
type mailerService struct {
	mailer *mailer.Mailer
}

// Yangi MailerService yaratish
func NewMailerService(m *mailer.Mailer) MailerService {
	return &mailerService{mailer: m}
}

// Send funksiyasi - Email yuborish
func (m *mailerService) Send(to, subject, body string) error {
	return m.mailer.Send(to, subject, body)
}

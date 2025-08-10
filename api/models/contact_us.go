package models

import "time"

type ContactMessage struct {
	ID        string     `json:"id"`
	Name      string     `json:"name"`
	Email     string     `json:"email"`
	Subject   *string    `json:"subject,omitempty"`
	Message   string     `json:"message"`
	IsRead    bool       `json:"is_read"`
	CreatedAt time.Time  `json:"created_at"`
	RepliedAt *time.Time `json:"replied_at,omitempty"`
}

type ContactCreateRequest struct {
	Name          string  `json:"name" binding:"required,min=2"`
	Email         string  `json:"email" binding:"required,email"`
	Subject       *string `json:"subject,omitempty"`
	Message       string  `json:"message" binding:"required,min=5"`
	TermsAccepted bool    `json:"terms_accepted"` // true bo'lishi kerak (biznes qoidasi)
}

type ContactCreateResponse struct {
	ID string `json:"id"`
}

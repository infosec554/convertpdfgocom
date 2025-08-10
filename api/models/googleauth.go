package models

type GoogleAuthRequest struct {
	Code string `json:"code" binding:"required" example:"4/0AX4XfW..."` // Google authorization code
}

type GoogleUser struct {
	Email    string `json:"email"`
	Name     string `json:"name"`
	GoogleID string `json:"google_id"`
	Picture  string `json:"picture,omitempty"`
}

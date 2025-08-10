package models

type FacebookUser struct {
	FacebookID string `json:"facebook_id"`
	Name       string `json:"name"`
	Email      string `json:"email"`
	Picture    string `json:"picture,omitempty"`
}

type FacebookAuthRequest struct {
	Code string `json:"code" binding:"required" example:"AQDt1n..."` // Facebookdan kelgan code
}

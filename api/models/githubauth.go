package models

type GithubUser struct {
	Username string
	Email    string
	GithubID string
	Name     string
}
type GithubAuthRequest struct {
    Code string `json:"code" binding:"required" example:"AQDt1n..."` 
}

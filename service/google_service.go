package service

import (
	"context"
	"fmt"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	oauth2v2 "google.golang.org/api/oauth2/v2"

	"convertpdfgo/api/models"
)

type GoogleOAuthConfig struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string
}

type GoogleService interface {
	ExchangeCodeForUser(ctx context.Context, code string) (*models.GoogleUser, error)
}

type googleService struct {
	oauthConfig *oauth2.Config
}

func NewGoogleService(config GoogleOAuthConfig) GoogleService {
	return &googleService{
		oauthConfig: &oauth2.Config{
			ClientID:     config.ClientID,
			ClientSecret: config.ClientSecret,
			RedirectURL:  config.RedirectURL,
			Scopes: []string{
				"https://www.googleapis.com/auth/userinfo.email",
				"https://www.googleapis.com/auth/userinfo.profile",
			},
			Endpoint: google.Endpoint,
		},
	}
}

func (g *googleService) ExchangeCodeForUser(ctx context.Context, code string) (*models.GoogleUser, error) {
	token, err := g.oauthConfig.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code for token: %w", err)
	}

	oauth2Service, err := oauth2v2.New(g.oauthConfig.Client(ctx, token))
	if err != nil {
		return nil, fmt.Errorf("failed to create oauth2 client: %w", err)
	}

	userinfo, err := oauth2Service.Userinfo.Get().Do()
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}

	return &models.GoogleUser{
		Email:    userinfo.Email,
		Name:     userinfo.Name,
		GoogleID: userinfo.Id,
	}, nil
}

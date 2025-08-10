package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"convertpdfgo/api/models"
)

type FacebookOAuthConfig struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string
}

type FacebookService interface {
	ExchangeCodeForUser(ctx context.Context, code string) (*models.FacebookUser, error)
}

type facebookService struct {
	config FacebookOAuthConfig
}

func NewFacebookService(cfg FacebookOAuthConfig) FacebookService {
	return &facebookService{config: cfg}
}

func (f *facebookService) ExchangeCodeForUser(ctx context.Context, code string) (*models.FacebookUser, error) {
	// 1. code ni access_token ga aylantirish
	tokenURL := "https://graph.facebook.com/v18.0/oauth/access_token"
	data := url.Values{}
	data.Set("client_id", f.config.ClientID)
	data.Set("client_secret", f.config.ClientSecret)
	data.Set("redirect_uri", f.config.RedirectURL)
	data.Set("code", code)

	req, _ := http.NewRequestWithContext(ctx, "GET", tokenURL+"?"+data.Encode(), nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var tok struct {
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tok); err != nil {
		return nil, fmt.Errorf("failed to decode token: %w", err)
	}
	if tok.AccessToken == "" {
		return nil, fmt.Errorf("facebook did not return access_token")
	}

	// 2. access_token orqali userni olish (fields= id, name, email, picture)
	userInfoURL := fmt.Sprintf("https://graph.facebook.com/me?fields=id,name,email,picture.type(large)&access_token=%s", tok.AccessToken)
	userReq, _ := http.NewRequestWithContext(ctx, "GET", userInfoURL, nil)
	userResp, err := http.DefaultClient.Do(userReq)
	if err != nil {
		return nil, err
	}
	defer userResp.Body.Close()

	var fb struct {
		ID      string `json:"id"`
		Name    string `json:"name"`
		Email   string `json:"email"`
		Picture struct {
			Data struct {
				Url string `json:"url"`
			} `json:"data"`
		} `json:"picture"`
	}
	if err := json.NewDecoder(userResp.Body).Decode(&fb); err != nil {
		return nil, fmt.Errorf("failed to decode user info: %w", err)
	}

	return &models.FacebookUser{
		FacebookID: fb.ID,
		Name:       fb.Name,
		Email:      fb.Email,
		Picture:    fb.Picture.Data.Url,
	}, nil
}

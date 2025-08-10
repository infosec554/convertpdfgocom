package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"convertpdfgo/api/models"
)

type GithubOAuthConfig struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string
}

type GithubService interface {
	ExchangeCodeForUser(ctx context.Context, code string) (*models.GithubUser, error)
}

type githubService struct {
	config GithubOAuthConfig
}

func NewGithubService(cfg GithubOAuthConfig) GithubService {
	return &githubService{config: cfg}
}

func (g *githubService) ExchangeCodeForUser(ctx context.Context, code string) (*models.GithubUser, error) {
	// 1. code ni access_token ga aylantirish
	tokenURL := "https://github.com/login/oauth/access_token"
	data := url.Values{}
	data.Set("client_id", g.config.ClientID)
	data.Set("client_secret", g.config.ClientSecret)
	data.Set("code", code)
	data.Set("redirect_uri", g.config.RedirectURL)

	req, _ := http.NewRequestWithContext(ctx, "POST", tokenURL, strings.NewReader(data.Encode()))
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var tok struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tok); err != nil {
		return nil, err
	}
	if tok.AccessToken == "" {
		return nil, fmt.Errorf("github did not return access_token")
	}

	// 2. access_token orqali userni olish
	userReq, _ := http.NewRequestWithContext(ctx, "GET", "https://api.github.com/user", nil)
	userReq.Header.Set("Authorization", "token "+tok.AccessToken)
	userResp, err := http.DefaultClient.Do(userReq)
	if err != nil {
		return nil, err
	}
	defer userResp.Body.Close()

	var ghUser struct {
		Login string `json:"login"`
		ID    int    `json:"id"`
		Name  string `json:"name"`
		Email string `json:"email"`
	}
	if err := json.NewDecoder(userResp.Body).Decode(&ghUser); err != nil {
		return nil, err
	}

	return &models.GithubUser{
		Username: ghUser.Login,
		Email:    ghUser.Email,
		GithubID: fmt.Sprintf("%d", ghUser.ID),
		Name:     ghUser.Name,
	}, nil
}

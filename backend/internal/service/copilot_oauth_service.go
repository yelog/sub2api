package service

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/copilot"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/httpclient"
)

type CopilotOAuthStatus string

const (
	CopilotOAuthStatusPending  CopilotOAuthStatus = "pending"
	CopilotOAuthStatusComplete CopilotOAuthStatus = "complete"
)

type CopilotStartDeviceFlowInput struct {
	ProxyID *int64
}

type CopilotDeviceFlowSession struct {
	DeviceCode      string
	UserCode        string
	VerificationURI string
	ExpiresAt       time.Time
	Interval        int
}

type CopilotPollDeviceFlowInput struct {
	DeviceCode string
	ProxyID    *int64
}

type CopilotOAuthResult struct {
	Status      CopilotOAuthStatus
	Message     string
	Credentials map[string]any
	GitHubLogin string
	GitHubName  string
	GitHubID    int64
}

type CopilotOAuthService struct {
	proxyRepo ProxyRepository

	deviceCodeURL  string
	accessTokenURL string
	githubUserURL  string
	tokenURL       string
	httpClient     *http.Client
}

func NewCopilotOAuthService(proxyRepo ProxyRepository) *CopilotOAuthService {
	return &CopilotOAuthService{
		proxyRepo:      proxyRepo,
		deviceCodeURL:  copilot.DeviceCodeURL,
		accessTokenURL: copilot.AccessTokenURL,
		githubUserURL:  copilot.GitHubUserURL,
		tokenURL:       copilot.TokenExchangeURL,
	}
}

func (s *CopilotOAuthService) StartDeviceFlow(ctx context.Context, input CopilotStartDeviceFlowInput) (*CopilotDeviceFlowSession, error) {
	client, err := s.clientForProxy(ctx, input.ProxyID)
	if err != nil {
		return nil, err
	}
	resp, err := copilot.RequestDeviceCode(ctx, client, s.deviceCodeURL)
	if err != nil {
		return nil, infraerrors.New(http.StatusBadGateway, "COPILOT_DEVICE_CODE_FAILED", fmt.Sprintf("request GitHub device code: %v", err))
	}
	interval := resp.Interval
	if interval <= 0 {
		interval = 5
	}
	return &CopilotDeviceFlowSession{
		DeviceCode:      resp.DeviceCode,
		UserCode:        resp.UserCode,
		VerificationURI: resp.VerificationURI,
		ExpiresAt:       time.Now().Add(time.Duration(resp.ExpiresIn) * time.Second),
		Interval:        interval,
	}, nil
}

func (s *CopilotOAuthService) PollDeviceFlow(ctx context.Context, input CopilotPollDeviceFlowInput) (*CopilotOAuthResult, error) {
	deviceCode := strings.TrimSpace(input.DeviceCode)
	if deviceCode == "" {
		return nil, infraerrors.BadRequest("COPILOT_DEVICE_CODE_REQUIRED", "device_code is required")
	}
	client, err := s.clientForProxy(ctx, input.ProxyID)
	if err != nil {
		return nil, err
	}
	tokenResp, err := copilot.PollAccessToken(ctx, client, s.accessTokenURL, deviceCode)
	if err != nil {
		return nil, infraerrors.New(http.StatusBadGateway, "COPILOT_ACCESS_TOKEN_FAILED", fmt.Sprintf("poll GitHub access token: %v", err))
	}
	if tokenResp.Error != "" {
		switch tokenResp.Error {
		case "authorization_pending":
			return &CopilotOAuthResult{Status: CopilotOAuthStatusPending, Message: "Waiting for GitHub authorization."}, nil
		case "slow_down":
			return &CopilotOAuthResult{Status: CopilotOAuthStatusPending, Message: "Polling too fast; slow down and retry later."}, nil
		case "expired_token":
			return nil, infraerrors.BadRequest("COPILOT_DEVICE_CODE_EXPIRED", "device code expired")
		case "access_denied":
			return nil, infraerrors.BadRequest("COPILOT_ACCESS_DENIED", "GitHub authorization was denied")
		default:
			msg := tokenResp.ErrorDesc
			if msg == "" {
				msg = tokenResp.Error
			}
			return nil, infraerrors.BadRequest("COPILOT_OAUTH_ERROR", msg)
		}
	}
	if strings.TrimSpace(tokenResp.AccessToken) == "" {
		return nil, infraerrors.New(http.StatusBadGateway, "COPILOT_EMPTY_ACCESS_TOKEN", "GitHub returned an empty access token")
	}

	user, err := copilot.GetGitHubUser(ctx, client, s.githubUserURL, tokenResp.AccessToken)
	if err != nil {
		return nil, infraerrors.New(http.StatusBadGateway, "COPILOT_GITHUB_USER_FAILED", fmt.Sprintf("fetch GitHub user: %v", err))
	}
	copilotToken, err := copilot.ExchangeToken(ctx, client, s.tokenURL, tokenResp.AccessToken)
	if err != nil {
		return nil, infraerrors.BadRequest("COPILOT_TOKEN_EXCHANGE_FAILED", fmt.Sprintf("GitHub token is valid but Copilot access is unavailable: %v", err))
	}
	credentials := s.BuildAccountCredentials(tokenResp.AccessToken, copilotToken, user)
	return &CopilotOAuthResult{
		Status:      CopilotOAuthStatusComplete,
		Message:     "GitHub Copilot authorization complete.",
		Credentials: credentials,
		GitHubLogin: user.Login,
		GitHubName:  user.Name,
		GitHubID:    user.ID,
	}, nil
}

func (s *CopilotOAuthService) BuildAccountCredentials(githubToken string, token *copilot.CopilotToken, user *copilot.GitHubUser) map[string]any {
	credentials := map[string]any{
		"github_access_token": githubToken,
	}
	if token != nil {
		credentials["copilot_token"] = token.Token
		credentials["copilot_token_expires_at"] = token.ExpiresAt.Unix()
		credentials["copilot_token_refresh_at"] = token.RefreshAt.Unix()
	}
	if user != nil {
		credentials["github_login"] = user.Login
		credentials["github_name"] = user.Name
		credentials["github_id"] = user.ID
		credentials["github_avatar_url"] = user.AvatarURL
	}
	return credentials
}

func (s *CopilotOAuthService) clientForProxy(ctx context.Context, proxyID *int64) (*http.Client, error) {
	if s.httpClient != nil && proxyID == nil {
		return s.httpClient, nil
	}
	proxyURL := ""
	if proxyID != nil {
		if s.proxyRepo == nil {
			return nil, infraerrors.BadRequest("COPILOT_PROXY_REPO_UNAVAILABLE", "proxy repository is unavailable")
		}
		proxy, err := s.proxyRepo.GetByID(ctx, *proxyID)
		if err != nil {
			return nil, err
		}
		if proxy != nil {
			proxyURL = proxy.URL()
		}
	}
	client, err := httpclient.GetClient(httpclient.Options{
		ProxyURL:              proxyURL,
		Timeout:               30 * time.Second,
		ResponseHeaderTimeout: 30 * time.Second,
	})
	if err != nil {
		return nil, infraerrors.New(http.StatusBadGateway, "COPILOT_HTTP_CLIENT_FAILED", fmt.Sprintf("create HTTP client: %v", err))
	}
	return client, nil
}

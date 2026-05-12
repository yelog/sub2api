package copilot

import "time"

const (
	DeviceOAuthClientID = "Iv1.b507a08c87ecfe98"
	DeviceCodeURL       = "https://github.com/login/device/code"
	AccessTokenURL      = "https://github.com/login/oauth/access_token"
	GitHubUserURL       = "https://api.github.com/user"
	InternalUserURL     = "https://api.github.com/copilot_internal/user"
	TokenExchangeURL    = "https://api.github.com/copilot_internal/v2/token"

	DefaultEditorVersion        = "vscode/1.98.1"
	DefaultEditorPluginVersion  = "copilot-chat/0.26.7"
	DefaultUserAgent            = "GitHubCopilotChat/0.26.7"
	DefaultGitHubAPIVersion     = "2025-04-01"
	defaultTokenExchangeRefresh = 30 * time.Minute
)

type DeviceCodeResponse struct {
	DeviceCode      string `json:"device_code"`
	UserCode        string `json:"user_code"`
	VerificationURI string `json:"verification_uri"`
	ExpiresIn       int    `json:"expires_in"`
	Interval        int    `json:"interval"`
}

type AccessTokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	Scope       string `json:"scope"`
	Error       string `json:"error,omitempty"`
	ErrorDesc   string `json:"error_description,omitempty"`
	Interval    int    `json:"interval,omitempty"`
}

type GitHubUser struct {
	Login     string `json:"login"`
	ID        int64  `json:"id"`
	AvatarURL string `json:"avatar_url"`
	Name      string `json:"name"`
}

type tokenExchangeResponse struct {
	Token        string `json:"token"`
	ExpiresAt    int64  `json:"expires_at"`
	RefreshIn    int64  `json:"refresh_in"`
	ErrorMessage string `json:"error_description,omitempty"`
}

type CopilotToken struct {
	Token     string
	ExpiresAt time.Time
	RefreshAt time.Time
}

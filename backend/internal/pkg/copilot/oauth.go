package copilot

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

func RequestDeviceCode(ctx context.Context, httpClient *http.Client, endpoint string) (*DeviceCodeResponse, error) {
	if endpoint == "" {
		endpoint = DeviceCodeURL
	}
	client := withDefaultClient(httpClient)
	form := url.Values{}
	form.Set("client_id", DeviceOAuthClientID)
	form.Set("scope", "read:user")

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("build device code request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	var result DeviceCodeResponse
	if err := doJSON(client, req, http.StatusOK, &result); err != nil {
		return nil, fmt.Errorf("request device code: %w", err)
	}
	return &result, nil
}

func PollAccessToken(ctx context.Context, httpClient *http.Client, endpoint, deviceCode string) (*AccessTokenResponse, error) {
	if endpoint == "" {
		endpoint = AccessTokenURL
	}
	client := withDefaultClient(httpClient)
	form := url.Values{}
	form.Set("client_id", DeviceOAuthClientID)
	form.Set("device_code", deviceCode)
	form.Set("grant_type", "urn:ietf:params:oauth:grant-type:device_code")

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("build access token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response body: %w", err)
	}
	var result AccessTokenResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("decode response body: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}
	return &result, nil
}

func GetGitHubUser(ctx context.Context, httpClient *http.Client, endpoint, accessToken string) (*GitHubUser, error) {
	if endpoint == "" {
		endpoint = GitHubUserURL
	}
	client := withDefaultClient(httpClient)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("build github user request: %w", err)
	}
	req.Header.Set("Authorization", "token "+accessToken)
	req.Header.Set("Accept", "application/json")

	var result GitHubUser
	if err := doJSON(client, req, http.StatusOK, &result); err != nil {
		return nil, fmt.Errorf("get github user: %w", err)
	}
	return &result, nil
}

func ExchangeToken(ctx context.Context, httpClient *http.Client, endpoint, githubToken string) (*CopilotToken, error) {
	if endpoint == "" {
		endpoint = TokenExchangeURL
	}
	client := withDefaultClient(httpClient)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("build copilot token request: %w", err)
	}
	req.Header.Set("Authorization", "token "+githubToken)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("editor-version", DefaultEditorVersion)
	req.Header.Set("editor-plugin-version", DefaultEditorPluginVersion)
	req.Header.Set("User-Agent", DefaultUserAgent)
	req.Header.Set("x-github-api-version", DefaultGitHubAPIVersion)
	req.Header.Set("x-vscode-user-agent-library-version", "electron-fetch")

	var result tokenExchangeResponse
	if err := doJSON(client, req, http.StatusOK, &result); err != nil {
		return nil, fmt.Errorf("exchange copilot token: %w", err)
	}
	if result.Token == "" {
		msg := result.ErrorMessage
		if msg == "" {
			msg = "empty token in response"
		}
		return nil, fmt.Errorf("exchange copilot token: %s", msg)
	}

	now := time.Now()
	expiresAt := now.Add(defaultTokenExchangeRefresh)
	if result.ExpiresAt > 0 {
		expiresAt = time.Unix(result.ExpiresAt, 0)
	}
	refreshIn := result.RefreshIn
	if refreshIn <= 0 {
		refreshIn = int64(time.Until(expiresAt).Seconds())
	}
	refreshAt := now.Add(time.Duration(refreshIn-60) * time.Second)
	if refreshAt.Before(now) {
		refreshAt = now.Add(30 * time.Second)
	}
	return &CopilotToken{Token: result.Token, ExpiresAt: expiresAt, RefreshAt: refreshAt}, nil
}

func withDefaultClient(client *http.Client) *http.Client {
	if client != nil {
		return client
	}
	return &http.Client{Timeout: 30 * time.Second}
}

func doJSON(client *http.Client, req *http.Request, expectedStatus int, out any) error {
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response body: %w", err)
	}
	if resp.StatusCode != expectedStatus {
		return fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}
	if err := json.Unmarshal(body, out); err != nil {
		return fmt.Errorf("decode response body: %w", err)
	}
	return nil
}

package service

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestCopilotOAuthService_StartDeviceFlow(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/device" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"device_code":      "device-1",
			"user_code":        "ABCD-EFGH",
			"verification_uri": "https://github.com/login/device",
			"expires_in":       900,
			"interval":         5,
		})
	}))
	defer server.Close()

	svc := NewCopilotOAuthService(nil)
	svc.httpClient = server.Client()
	svc.deviceCodeURL = server.URL + "/device"

	session, err := svc.StartDeviceFlow(context.Background(), CopilotStartDeviceFlowInput{})
	if err != nil {
		t.Fatal(err)
	}
	if session.DeviceCode != "device-1" || session.UserCode != "ABCD-EFGH" {
		t.Fatalf("unexpected session: %+v", session)
	}
	if session.Interval != 5 {
		t.Fatalf("interval = %d", session.Interval)
	}
	if !session.ExpiresAt.After(time.Now()) {
		t.Fatalf("expires_at not in future: %v", session.ExpiresAt)
	}
}

func TestCopilotOAuthService_PollDeviceFlowPending(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"error": "authorization_pending"})
	}))
	defer server.Close()

	svc := NewCopilotOAuthService(nil)
	svc.httpClient = server.Client()
	svc.accessTokenURL = server.URL

	result, err := svc.PollDeviceFlow(context.Background(), CopilotPollDeviceFlowInput{DeviceCode: "device-1"})
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != CopilotOAuthStatusPending {
		t.Fatalf("status = %s", result.Status)
	}
}

func TestCopilotOAuthService_PollDeviceFlowSuccess(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/access_token":
			_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "github-token", "token_type": "bearer"})
		case "/user":
			if got := r.Header.Get("Authorization"); got != "token github-token" {
				t.Fatalf("Authorization = %q", got)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"login": "octo", "id": 42, "name": "Octo Cat", "avatar_url": "https://example.com/a.png"})
		case "/copilot_token":
			_ = json.NewEncoder(w).Encode(map[string]any{"token": "copilot-token", "expires_at": time.Now().Add(time.Hour).Unix(), "refresh_in": 600})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	svc := NewCopilotOAuthService(nil)
	svc.httpClient = server.Client()
	svc.accessTokenURL = server.URL + "/access_token"
	svc.githubUserURL = server.URL + "/user"
	svc.tokenURL = server.URL + "/copilot_token"

	result, err := svc.PollDeviceFlow(context.Background(), CopilotPollDeviceFlowInput{DeviceCode: "device-1"})
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != CopilotOAuthStatusComplete {
		t.Fatalf("status = %s", result.Status)
	}
	if result.GitHubLogin != "octo" || result.GitHubID != 42 {
		t.Fatalf("unexpected user: %+v", result)
	}
	if got := result.Credentials["github_access_token"]; got != "github-token" {
		t.Fatalf("github_access_token = %v", got)
	}
	if got := result.Credentials["copilot_token"]; got != "copilot-token" {
		t.Fatalf("copilot_token = %v", got)
	}
}

func TestCopilotOAuthService_PollDeviceFlowExchangeFailureDoesNotLeakToken(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/access_token":
			_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "secret-github-token", "token_type": "bearer"})
		case "/user":
			_ = json.NewEncoder(w).Encode(map[string]any{"login": "octo", "id": 42})
		case "/copilot_token":
			http.Error(w, "no copilot", http.StatusForbidden)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	svc := NewCopilotOAuthService(nil)
	svc.httpClient = server.Client()
	svc.accessTokenURL = server.URL + "/access_token"
	svc.githubUserURL = server.URL + "/user"
	svc.tokenURL = server.URL + "/copilot_token"

	_, err := svc.PollDeviceFlow(context.Background(), CopilotPollDeviceFlowInput{DeviceCode: "device-1"})
	if err == nil {
		t.Fatal("expected error")
	}
	if strings.Contains(err.Error(), "secret-github-token") {
		t.Fatalf("error leaked token: %v", err)
	}
}

package copilot

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestRequestDeviceCodeSendsExpectedForm(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Accept"); got != "application/json" {
			t.Fatalf("Accept = %q", got)
		}
		if err := r.ParseForm(); err != nil {
			t.Fatal(err)
		}
		if got := r.Form.Get("client_id"); got != DeviceOAuthClientID {
			t.Fatalf("client_id = %q", got)
		}
		if got := r.Form.Get("scope"); got != "read:user" {
			t.Fatalf("scope = %q", got)
		}
		_ = json.NewEncoder(w).Encode(DeviceCodeResponse{
			DeviceCode:      "device-1",
			UserCode:        "ABCD-EFGH",
			VerificationURI: "https://github.com/login/device",
			ExpiresIn:       900,
			Interval:        5,
		})
	}))
	defer server.Close()

	resp, err := RequestDeviceCode(context.Background(), server.Client(), server.URL)
	if err != nil {
		t.Fatal(err)
	}
	if resp.DeviceCode != "device-1" || resp.UserCode != "ABCD-EFGH" {
		t.Fatalf("unexpected response: %+v", resp)
	}
}

func TestPollAccessTokenParsesAuthorizationPending(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatal(err)
		}
		if got := r.Form.Get("grant_type"); got != "urn:ietf:params:oauth:grant-type:device_code" {
			t.Fatalf("grant_type = %q", got)
		}
		_ = json.NewEncoder(w).Encode(AccessTokenResponse{Error: "authorization_pending"})
	}))
	defer server.Close()

	resp, err := PollAccessToken(context.Background(), server.Client(), server.URL, "device-1")
	if err != nil {
		t.Fatal(err)
	}
	if resp.Error != "authorization_pending" {
		t.Fatalf("error = %q", resp.Error)
	}
}

func TestExchangeTokenSendsCopilotHeaders(t *testing.T) {
	t.Parallel()

	expiresAt := time.Now().Add(time.Hour).Unix()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "token github-token" {
			t.Fatalf("Authorization = %q", got)
		}
		for _, header := range []string{"editor-version", "editor-plugin-version", "x-github-api-version", "x-vscode-user-agent-library-version"} {
			if strings.TrimSpace(r.Header.Get(header)) == "" {
				t.Fatalf("missing header %s", header)
			}
		}
		_ = json.NewEncoder(w).Encode(tokenExchangeResponse{Token: "copilot-token", ExpiresAt: expiresAt, RefreshIn: 600})
	}))
	defer server.Close()

	token, err := ExchangeToken(context.Background(), server.Client(), server.URL, "github-token")
	if err != nil {
		t.Fatal(err)
	}
	if token.Token != "copilot-token" {
		t.Fatalf("token = %q", token.Token)
	}
	if token.ExpiresAt.Unix() != expiresAt {
		t.Fatalf("expires_at = %v", token.ExpiresAt)
	}
}

func TestExchangeTokenRejectsNonSuccess(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "no copilot", http.StatusForbidden)
	}))
	defer server.Close()

	_, err := ExchangeToken(context.Background(), server.Client(), server.URL, "github-token")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "403") {
		t.Fatalf("unexpected error: %v", err)
	}
}

package service

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/pkg/tlsfingerprint"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

type copilotTestHTTPUpstream struct {
	lastReq  *http.Request
	lastBody []byte
}

func (u *copilotTestHTTPUpstream) Do(req *http.Request, _ string, _ int64, _ int) (*http.Response, error) {
	body, _ := io.ReadAll(req.Body)
	u.lastReq = req
	u.lastBody = body
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
		Body:       io.NopCloser(strings.NewReader("data: {\"type\":\"message_stop\"}\n\n")),
	}, nil
}

func (u *copilotTestHTTPUpstream) DoWithTLS(req *http.Request, proxyURL string, accountID int64, accountConcurrency int, _ *tlsfingerprint.Profile) (*http.Response, error) {
	return u.Do(req, proxyURL, accountID, accountConcurrency)
}

func TestAccountTestService_TestAccountConnection_CopilotAppliesModelMapping(t *testing.T) {
	gin.SetMode(gin.TestMode)

	account := Account{
		ID:          7,
		Name:        "copilot-oauth",
		Platform:    PlatformCopilot,
		Type:        AccountTypeOAuth,
		Status:      StatusActive,
		Schedulable: true,
		Concurrency: 1,
		Credentials: map[string]any{
			"github_access_token": "github-token",
			"copilot_token":       "copilot-token",
			"model_mapping": map[string]any{
				"gpt-5.4": "claude-sonnet-4.5",
			},
		},
	}
	upstream := &copilotTestHTTPUpstream{}
	svc := &AccountTestService{
		accountRepo:  stubOpenAIAccountRepo{accounts: []Account{account}},
		httpUpstream: upstream,
	}

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/admin/accounts/7/test", bytes.NewReader(nil))

	err := svc.TestAccountConnection(c, account.ID, "gpt-5.4", "", AccountTestModeDefault)
	require.NoError(t, err)
	require.NotNil(t, upstream.lastReq)
	require.Equal(t, "claude-sonnet-4.5", gjson.GetBytes(upstream.lastBody, "model").String())
	require.Equal(t, "Bearer copilot-token", upstream.lastReq.Header.Get("Authorization"))
	require.Contains(t, rec.Body.String(), `"model":"claude-sonnet-4.5"`)
	require.Contains(t, rec.Body.String(), `"type":"test_complete"`)
}

func TestAccountTestService_TestAccountConnection_CopilotMissingToken(t *testing.T) {
	gin.SetMode(gin.TestMode)

	account := Account{
		ID:          8,
		Name:        "copilot-oauth-missing-token",
		Platform:    PlatformCopilot,
		Type:        AccountTypeOAuth,
		Status:      StatusActive,
		Schedulable: true,
		Concurrency: 1,
		Credentials: map[string]any{},
	}
	upstream := &copilotTestHTTPUpstream{}
	svc := &AccountTestService{
		accountRepo:  stubOpenAIAccountRepo{accounts: []Account{account}},
		httpUpstream: upstream,
	}

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/admin/accounts/8/test", bytes.NewReader(nil))

	err := svc.TestAccountConnection(c, account.ID, "gpt-5.4", "", AccountTestModeDefault)
	require.NoError(t, err)
	require.Nil(t, upstream.lastReq)
	require.Contains(t, rec.Body.String(), "No Copilot token available")
}

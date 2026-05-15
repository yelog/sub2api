//go:build unit

package service

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestOpenAIGatewayForwardAsChatCompletions_CopilotOAuthDirectPassthrough(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)

	resp := newJSONResponse(http.StatusOK, `{"id":"chatcmpl-test","model":"gpt-4o-2024-11-20","choices":[{"message":{"role":"assistant","content":"ok"},"finish_reason":"stop","index":0}],"usage":{"prompt_tokens":3,"completion_tokens":2,"total_tokens":5}}`)
	upstream := &queuedHTTPUpstream{responses: []*http.Response{resp}}
	svc := &OpenAIGatewayService{httpUpstream: upstream}
	account := &Account{
		ID:          401,
		Platform:    PlatformCopilot,
		Type:        AccountTypeOAuth,
		Concurrency: 1,
		Credentials: map[string]any{
			"github_access_token":      "github-token",
			"copilot_token":            "copilot-token",
			"copilot_token_expires_at": time.Now().Add(time.Hour).Unix(),
		},
	}
	body := []byte(`{"model":"gpt-4o","messages":[{"role":"user","content":"hello"}],"stream":false}`)

	result, err := svc.ForwardAsChatCompletions(c.Request.Context(), c, account, body, "", "")

	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, "gpt-4o", result.Model)
	require.Len(t, upstream.requests, 1)
	require.Equal(t, "https://api.githubcopilot.com/chat/completions", upstream.requests[0].URL.String())
	require.Equal(t, "Bearer copilot-token", upstream.requests[0].Header.Get("Authorization"))
	require.NotEmpty(t, upstream.requests[0].Header.Get("Editor-Version"))
	require.NotEmpty(t, upstream.requests[0].Header.Get("Editor-Plugin-Version"))
	require.NotEmpty(t, upstream.requests[0].Header.Get("X-GitHub-Api-Version"))
	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), "ok")
}

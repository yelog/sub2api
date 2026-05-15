package service

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/copilot"
	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
	"go.uber.org/zap"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
)

const copilotChatCompletionsURL = "https://api.githubcopilot.com/chat/completions"

func (s *OpenAIGatewayService) forwardCopilotChatCompletions(
	ctx context.Context,
	c *gin.Context,
	account *Account,
	body []byte,
	defaultMappedModel string,
) (*OpenAIForwardResult, error) {
	startTime := time.Now()

	originalModel := gjson.GetBytes(body, "model").String()
	if originalModel == "" {
		writeChatCompletionsError(c, http.StatusBadRequest, "invalid_request_error", "model is required")
		return nil, fmt.Errorf("missing model in request")
	}
	clientStream := gjson.GetBytes(body, "stream").Bool()
	reasoningEffort := extractOpenAIReasoningEffortFromBody(body, originalModel)
	serviceTier := extractOpenAIServiceTierFromBody(body)

	billingModel := strings.TrimSpace(defaultMappedModel)
	if billingModel == "" {
		billingModel = account.GetMappedModel(originalModel)
	}
	if billingModel == "" {
		billingModel = originalModel
	}
	upstreamModel := billingModel

	upstreamBody := body
	if upstreamModel != originalModel {
		upstreamBody = ReplaceModelInBody(body, upstreamModel)
	}
	if clientStream {
		var usageErr error
		upstreamBody, usageErr = ensureOpenAIChatStreamUsage(upstreamBody)
		if usageErr != nil {
			return nil, fmt.Errorf("enable stream usage: %w", usageErr)
		}
	}

	token, err := s.getUsableCopilotToken(ctx, account)
	if err != nil {
		return nil, fmt.Errorf("get copilot access token: %w", err)
	}
	if token == "" {
		return nil, errors.New("copilot access token not found in credentials")
	}

	logger.L().Debug("copilot chat_completions: forwarding raw chat completions",
		zap.Int64("account_id", account.ID),
		zap.String("original_model", originalModel),
		zap.String("upstream_model", upstreamModel),
		zap.Bool("stream", clientStream),
	)

	upstreamCtx, releaseUpstreamCtx := detachUpstreamContext(ctx)
	upstreamReq, err := http.NewRequestWithContext(upstreamCtx, http.MethodPost, copilotChatCompletionsURL, bytes.NewReader(upstreamBody))
	releaseUpstreamCtx()
	if err != nil {
		return nil, fmt.Errorf("build copilot upstream request: %w", err)
	}
	applyCopilotChatCompletionsHeaders(upstreamReq, token, clientStream)

	proxyURL := ""
	if account.ProxyID != nil && account.Proxy != nil {
		proxyURL = account.Proxy.URL()
	}

	resp, err := s.httpUpstream.DoWithTLS(upstreamReq, proxyURL, account.ID, account.Concurrency, nil)
	if err != nil {
		safeErr := sanitizeUpstreamErrorMessage(err.Error())
		setOpsUpstreamError(c, 0, safeErr, "")
		appendOpsUpstreamError(c, OpsUpstreamErrorEvent{
			Platform:           account.Platform,
			AccountID:          account.ID,
			AccountName:        account.Name,
			UpstreamStatusCode: 0,
			Kind:               "request_error",
			Message:            safeErr,
		})
		writeChatCompletionsError(c, http.StatusBadGateway, "upstream_error", "Upstream request failed")
		return nil, fmt.Errorf("copilot upstream request failed: %s", safeErr)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
		_ = resp.Body.Close()
		resp.Body = io.NopCloser(bytes.NewReader(respBody))
		upstreamMsg := sanitizeUpstreamErrorMessage(strings.TrimSpace(extractUpstreamErrorMessage(respBody)))
		if s.shouldFailoverOpenAIUpstreamResponse(resp.StatusCode, upstreamMsg, respBody) {
			appendOpsUpstreamError(c, OpsUpstreamErrorEvent{
				Platform:           account.Platform,
				AccountID:          account.ID,
				AccountName:        account.Name,
				UpstreamStatusCode: resp.StatusCode,
				UpstreamRequestID:  resp.Header.Get("x-request-id"),
				Kind:               "failover",
				Message:            upstreamMsg,
			})
			if s.rateLimitService != nil {
				s.rateLimitService.HandleUpstreamError(ctx, account, resp.StatusCode, resp.Header, respBody)
			}
			return nil, &UpstreamFailoverError{StatusCode: resp.StatusCode, ResponseBody: respBody}
		}
		return s.handleChatCompletionsErrorResponse(resp, c, account)
	}

	if clientStream {
		return s.streamRawChatCompletions(c, resp, originalModel, billingModel, upstreamModel, reasoningEffort, serviceTier, startTime)
	}
	return s.bufferRawChatCompletions(c, resp, originalModel, billingModel, upstreamModel, reasoningEffort, serviceTier, startTime)
}

func (s *OpenAIGatewayService) getUsableCopilotToken(ctx context.Context, account *Account) (string, error) {
	copilotToken := strings.TrimSpace(account.GetCredential("copilot_token"))
	if copilotToken != "" && !isCopilotTokenRefreshDue(account) {
		return copilotToken, nil
	}

	githubToken := strings.TrimSpace(account.GetCredential("github_access_token"))
	if githubToken == "" {
		if copilotToken != "" && !isCopilotTokenExpired(account) {
			return copilotToken, nil
		}
		return "", errors.New("github_access_token not found in credentials")
	}

	endpoint := copilot.TokenExchangeURL
	refreshed, err := copilot.ExchangeToken(ctx, nil, endpoint, githubToken)
	if err != nil {
		if copilotToken != "" && !isCopilotTokenExpired(account) {
			return copilotToken, nil
		}
		return "", err
	}

	credentials := cloneCredentials(account.Credentials)
	credentials["copilot_token"] = refreshed.Token
	credentials["copilot_token_expires_at"] = refreshed.ExpiresAt.Unix()
	credentials["copilot_token_refresh_at"] = refreshed.RefreshAt.Unix()
	if err := persistAccountCredentials(ctx, s.accountRepo, account, credentials); err != nil {
		return "", err
	}
	return refreshed.Token, nil
}

func isCopilotTokenRefreshDue(account *Account) bool {
	refreshAt := account.GetCredentialAsTime("copilot_token_refresh_at")
	if refreshAt != nil {
		return time.Now().After(*refreshAt)
	}
	return isCopilotTokenExpired(account)
}

func isCopilotTokenExpired(account *Account) bool {
	expiresAt := account.GetCredentialAsTime("copilot_token_expires_at")
	if expiresAt == nil {
		return false
	}
	return time.Now().Add(60 * time.Second).After(*expiresAt)
}

func applyCopilotChatCompletionsHeaders(req *http.Request, token string, stream bool) {
	req.Header.Set("Content-Type", "application/json")
	if stream {
		req.Header.Set("Accept", "text/event-stream")
	} else {
		req.Header.Set("Accept", "application/json")
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Editor-Version", copilot.DefaultEditorVersion)
	req.Header.Set("Editor-Plugin-Version", copilot.DefaultEditorPluginVersion)
	req.Header.Set("User-Agent", copilot.DefaultUserAgent)
	req.Header.Set("X-GitHub-Api-Version", copilot.DefaultGitHubAPIVersion)
	req.Header.Set("Openai-Intent", "conversation-edits")
	req.Header.Set("x-initiator", "user")
}

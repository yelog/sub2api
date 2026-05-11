package admin

import (
	"math"
	"net/http"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type CopilotOAuthHandler struct {
	copilotOAuthService *service.CopilotOAuthService
}

func NewCopilotOAuthHandler(copilotOAuthService *service.CopilotOAuthService) *CopilotOAuthHandler {
	return &CopilotOAuthHandler{copilotOAuthService: copilotOAuthService}
}

type CopilotStartOAuthRequest struct {
	ProxyID *int64 `json:"proxy_id"`
}

type CopilotPollOAuthRequest struct {
	DeviceCode string `json:"device_code" binding:"required"`
	ProxyID    *int64 `json:"proxy_id"`
}

func (h *CopilotOAuthHandler) StartDeviceFlow(c *gin.Context) {
	var req CopilotStartOAuthRequest
	if c.Request.Body != nil && c.Request.ContentLength != 0 {
		if err := c.ShouldBindJSON(&req); err != nil {
			response.BadRequest(c, "Invalid request: "+err.Error())
			return
		}
	}
	session, err := h.copilotOAuthService.StartDeviceFlow(c.Request.Context(), service.CopilotStartDeviceFlowInput{ProxyID: req.ProxyID})
	if err != nil {
		respondCopilotOAuthError(c, err)
		return
	}
	expiresIn := int(math.Max(0, time.Until(session.ExpiresAt).Seconds()))
	response.Success(c, gin.H{
		"device_code":      session.DeviceCode,
		"user_code":        session.UserCode,
		"verification_uri": session.VerificationURI,
		"expires_in":       expiresIn,
		"interval":         session.Interval,
	})
}

func (h *CopilotOAuthHandler) PollDeviceFlow(c *gin.Context) {
	var req CopilotPollOAuthRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	result, err := h.copilotOAuthService.PollDeviceFlow(c.Request.Context(), service.CopilotPollDeviceFlowInput{
		DeviceCode: req.DeviceCode,
		ProxyID:    req.ProxyID,
	})
	if err != nil {
		respondCopilotOAuthError(c, err)
		return
	}
	payload := gin.H{
		"status":  result.Status,
		"message": result.Message,
	}
	if result.Status == service.CopilotOAuthStatusComplete {
		payload["credentials"] = result.Credentials
		payload["github_login"] = result.GitHubLogin
		payload["github_name"] = result.GitHubName
		payload["github_id"] = result.GitHubID
	}
	response.Success(c, payload)
}

func respondCopilotOAuthError(c *gin.Context, err error) {
	status := infraerrors.Code(err)
	message := infraerrors.Message(err)
	if message == "" {
		message = err.Error()
	}
	if status >= http.StatusInternalServerError {
		response.InternalError(c, message)
		return
	}
	response.BadRequest(c, message)
}

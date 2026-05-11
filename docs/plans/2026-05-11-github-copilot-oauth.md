# GitHub Copilot OAuth Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add minimal GitHub Copilot Device OAuth login support to admin APIs without merging `PR #771` or changing database schema.

**Architecture:** Add a small `copilot` package for GitHub Device OAuth and Copilot token exchange, a service that orchestrates proxy-aware HTTP calls and credentials payload building, and an admin handler/route pair under `/api/v1/admin/copilot/oauth`. Reuse the existing `Account` `Platform`/`Type`/`Credentials` structure and existing admin route/DI patterns.

**Tech Stack:** Go, Gin, Wire dependency injection, existing `service.ProxyRepository`, existing `pkg/response`, Go `httptest` for unit tests.

---

## Task 1: Add Copilot platform constant

**Files:**
- Modify: `backend/internal/domain/constants.go`
- Modify: `backend/internal/domain/constants_test.go`

**Step 1: Write/adjust the constants test**

Add `PlatformCopilot` to the platform validation expectations near existing `PlatformOpenAI`, `PlatformGemini`, and `PlatformAntigravity` tests.

Expected constant value:

```go
PlatformCopilot = "copilot"
```

**Step 2: Run the focused constants test**

```bash
cd /data/workspace/sub2api/backend
go test ./internal/domain -run 'Platform|Constants' -count=1
```

Expected: fail until constant is added, or pass if no explicit test exists yet.

**Step 3: Add the constant**

In `backend/internal/domain/constants.go`, add:

```go
PlatformCopilot = "copilot"
```

near the other platform constants.

**Step 4: Run the test again**

```bash
cd /data/workspace/sub2api/backend
go test ./internal/domain -run 'Platform|Constants' -count=1
```

Expected: PASS.

---

## Task 2: Add package-level Copilot OAuth client

**Files:**
- Create: `backend/internal/pkg/copilot/types.go`
- Create: `backend/internal/pkg/copilot/oauth.go`
- Create: `backend/internal/pkg/copilot/oauth_test.go`

**Step 1: Write client tests with `httptest`**

Cover:

- device code request sends `client_id`, `scope=read:user`, `Accept: application/json`
- access token polling returns `authorization_pending` as parsed payload, not transport error
- token exchange sends required Copilot headers and rejects non-200

**Step 2: Implement minimal types**

Create types for:

```go
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

type CopilotToken struct {
    Token     string
    ExpiresAt time.Time
    RefreshAt time.Time
}
```

Constants:

```go
DeviceOAuthClientID = "Iv1.b507a08c87ecfe98"
DeviceCodeURL       = "https://github.com/login/device/code"
AccessTokenURL      = "https://github.com/login/oauth/access_token"
GitHubUserURL       = "https://api.github.com/user"
TokenExchangeURL    = "https://api.github.com/copilot_internal/v2/token"
```

**Step 3: Implement HTTP methods**

Functions:

```go
RequestDeviceCode(ctx context.Context, httpClient *http.Client, endpoint string) (*DeviceCodeResponse, error)
PollAccessToken(ctx context.Context, httpClient *http.Client, endpoint, deviceCode string) (*AccessTokenResponse, error)
GetGitHubUser(ctx context.Context, httpClient *http.Client, endpoint, accessToken string) (*GitHubUser, error)
ExchangeToken(ctx context.Context, httpClient *http.Client, endpoint, githubToken string) (*CopilotToken, error)
```

Use endpoint override for tests; service passes defaults.

**Step 4: Run package tests**

```bash
cd /data/workspace/sub2api/backend
go test ./internal/pkg/copilot -count=1
```

Expected: PASS.

---

## Task 3: Add Copilot OAuth service

**Files:**
- Create: `backend/internal/service/copilot_oauth_service.go`
- Create: `backend/internal/service/copilot_oauth_service_test.go`
- Modify: `backend/internal/service/wire.go`

**Step 1: Write service tests**

Use a local `httptest.Server` and endpoint overrides. Cover:

- `StartDeviceFlow` returns `device_code`, `user_code`, `verification_uri`, `expires_at`, `interval`
- `PollDeviceFlow` maps `authorization_pending` / `slow_down` to typed pending states
- success returns `Credentials` containing `github_access_token`, `copilot_token`, `copilot_token_expires_at`, `github_login`, `github_id`, `github_name`
- Copilot exchange failure returns a user-safe error and does not include token values

**Step 2: Implement service structs**

Core request/response types:

```go
type CopilotStartDeviceFlowInput struct { ProxyID *int64 }
type CopilotDeviceFlowSession struct { DeviceCode, UserCode, VerificationURI string; ExpiresAt time.Time; Interval int }
type CopilotPollDeviceFlowInput struct { DeviceCode string; ProxyID *int64 }
type CopilotOAuthStatus string
const (
    CopilotOAuthStatusPending CopilotOAuthStatus = "pending"
    CopilotOAuthStatusComplete CopilotOAuthStatus = "complete"
)
type CopilotOAuthResult struct { Status CopilotOAuthStatus; Message string; Credentials map[string]any; GitHubLogin string; GitHubName string; GitHubID int64 }
```

**Step 3: Implement proxy-aware HTTP client creation**

- Inject `ProxyRepository` into `NewCopilotOAuthService(proxyRepo ProxyRepository)`.
- If `proxy_id` is present, load via `proxyRepo.GetByID(ctx, id)` and use `proxy.URL()`.
- Use existing `internal/pkg/httpclient` if exported enough; otherwise use a minimal local `http.Transport{Proxy: http.ProxyURL(parsed)}`.
- Invalid proxy URLs should return an error.

**Step 4: Implement service methods**

Methods:

```go
StartDeviceFlow(ctx context.Context, input CopilotStartDeviceFlowInput) (*CopilotDeviceFlowSession, error)
PollDeviceFlow(ctx context.Context, input CopilotPollDeviceFlowInput) (*CopilotOAuthResult, error)
BuildAccountCredentials(githubToken string, token *copilot.CopilotToken, user *copilot.GitHubUser) map[string]any
```

`PollDeviceFlow` flow:

1. reject empty `device_code`
2. call GitHub access-token endpoint
3. if `authorization_pending` or `slow_down`, return pending result
4. if OAuth error, return bad request style error
5. fetch GitHub user
6. exchange Copilot token
7. return complete result with credentials

**Step 5: Register service in Wire provider set**

Add `NewCopilotOAuthService` to `backend/internal/service/wire.go` near other OAuth services.

**Step 6: Run service tests**

```bash
cd /data/workspace/sub2api/backend
go test ./internal/service -run CopilotOAuth -count=1
```

Expected: PASS.

---

## Task 4: Add admin handler and routes

**Files:**
- Create: `backend/internal/handler/admin/copilot_oauth_handler.go`
- Create: `backend/internal/handler/admin/copilot_oauth_handler_test.go`
- Modify: `backend/internal/handler/handler.go`
- Modify: `backend/internal/handler/wire.go`
- Modify: `backend/internal/server/routes/admin.go`

**Step 1: Write handler tests**

Cover:

- `POST /start` validates JSON and returns `device_code`, `user_code`, `verification_uri`, `expires_in`, `interval`
- `POST /poll` with pending result returns HTTP 200 and `status: pending`
- `POST /poll` with complete result returns HTTP 200 and `credentials`
- empty `device_code` returns 400

**Step 2: Implement handler**

Routes target:

- `POST /api/v1/admin/copilot/oauth/start`
- `POST /api/v1/admin/copilot/oauth/poll`

Request structs:

```go
type CopilotStartOAuthRequest struct { ProxyID *int64 `json:"proxy_id"` }
type CopilotPollOAuthRequest struct { DeviceCode string `json:"device_code" binding:"required"`; ProxyID *int64 `json:"proxy_id"` }
```

Do not log tokens. Use `response.Success`, `response.BadRequest`, and existing error helpers consistently with other admin handlers.

**Step 3: Add handler to aggregator**

- Add `CopilotOAuth *admin.CopilotOAuthHandler` to `AdminHandlers` in `backend/internal/handler/handler.go`.
- Add constructor argument and struct assignment in `ProvideAdminHandlers` in `backend/internal/handler/wire.go`.
- Add `admin.NewCopilotOAuthHandler` to handler `ProviderSet`.

**Step 4: Register routes**

In `backend/internal/server/routes/admin.go`:

```go
if h.CopilotOAuth != nil {
    registerCopilotOAuthRoutes(admin, h)
}
```

Add helper:

```go
func registerCopilotOAuthRoutes(admin *gin.RouterGroup, h *handler.AdminHandlers) {
    copilot := admin.Group("/copilot")
    oauth := copilot.Group("/oauth")
    oauth.POST("/start", h.CopilotOAuth.StartDeviceFlow)
    oauth.POST("/poll", h.CopilotOAuth.PollDeviceFlow)
}
```

**Step 5: Run handler/routes tests**

```bash
cd /data/workspace/sub2api/backend
go test ./internal/handler/admin -run CopilotOAuth -count=1
go test ./internal/server -run 'Contract|Admin|Route' -count=1
```

Expected: PASS.

---

## Task 5: Update generated Wire file manually or run Wire

**Files:**
- Modify: `backend/cmd/server/wire_gen.go`
- Possibly modify: `backend/cmd/server/wire_gen_test.go`

**Step 1: Try code generation if available**

```bash
cd /data/workspace/sub2api/backend
go generate ./cmd/server
```

Expected: updates `wire_gen.go`, or fails if `wire` is not installed.

**Step 2: If generation is unavailable, patch `wire_gen.go` manually**

Mirror existing Antigravity OAuth wiring:

- instantiate `copilotOAuthService := service.NewCopilotOAuthService(proxyRepository)`
- instantiate `copilotOAuthHandler := admin.NewCopilotOAuthHandler(copilotOAuthService)`
- pass handler to `handler.ProvideAdminHandlers(...)` at the matching new argument position

**Step 3: Compile server package**

```bash
cd /data/workspace/sub2api/backend
go test ./cmd/server -count=1
```

Expected: PASS.

---

## Task 6: Final verification and regression check

**Files:**
- No new files unless failures require fixes.

**Step 1: Run focused backend tests**

```bash
cd /data/workspace/sub2api/backend
go test ./internal/pkg/copilot ./internal/service ./internal/handler/admin ./internal/server ./cmd/server -count=1
```

Expected: PASS.

**Step 2: Run formatting**

```bash
cd /data/workspace/sub2api/backend
gofmt -w internal/pkg/copilot internal/service/copilot_oauth_service.go internal/handler/admin/copilot_oauth_handler.go internal/handler/handler.go internal/handler/wire.go internal/server/routes/admin.go cmd/server/wire_gen.go
```

**Step 3: Run focused tests again**

```bash
cd /data/workspace/sub2api/backend
go test ./internal/pkg/copilot ./internal/service ./internal/handler/admin ./internal/server ./cmd/server -count=1
```

Expected: PASS.

**Step 4: Inspect git diff**

```bash
cd /data/workspace/sub2api
git diff --stat
git diff -- backend/internal/domain/constants.go backend/internal/server/routes/admin.go backend/internal/handler/handler.go
```

Expected: only minimal Copilot OAuth changes plus docs.

**Step 5: Summarize**

Mention:

- new endpoints `POST /api/v1/admin/copilot/oauth/start` and `POST /api/v1/admin/copilot/oauth/poll`
- platform constant `PlatformCopilot`
- tests run and outcome
- note that `/dashboard/openai-token-stats`, `/antigravity/default-model-mapping`, and `/internal/server/routes/gateway.go` were not intentionally changed.

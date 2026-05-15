# Anthropic Messages OpenAI Dispatch Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Make Anthropic `/v1/messages` requests work when the selected account is OpenAI by routing through the existing Anthropic-to-OpenAI compatibility forwarder.

**Architecture:** Keep `/v1/messages` request parsing and scheduling in `GatewayHandler.Messages`. After account selection, branch on the selected account platform. OpenAI accounts call `OpenAIGatewayService.ForwardAsAnthropic`; existing platform paths remain unchanged.

**Tech Stack:** Go, Gin, Sub2API gateway services, existing OpenAI Responses compatibility layer, Docker deployment scripts.

---

### Task 1: Wire OpenAI compatibility service into GatewayHandler

**Files:**
- Modify: `backend/internal/handler/gateway_handler.go`
- Modify: handler construction call sites if needed under `backend/internal/handler` or service initialization files.

**Step 1: Add field**

Add an `openAIGatewayService *service.OpenAIGatewayService` field to `GatewayHandler`.

**Step 2: Add constructor parameter**

Update `NewGatewayHandler` to accept the OpenAI gateway service and store it.

**Step 3: Update call sites**

Find `NewGatewayHandler(...)` call sites and pass the existing OpenAI gateway service instance.

**Step 4: Build compile check**

Run:

```bash
go test ./internal/handler -run TestResolveOpenAIMessagesDispatchMappedModel -count=1
```

Expected: compile succeeds and test passes.

### Task 2: Add selected-account OpenAI dispatch branch

**Files:**
- Modify: `backend/internal/handler/gateway_handler.go`

**Step 1: Derive OpenAI mapped model**

Before the forwarding branch, compute the default mapped model for OpenAI messages dispatch from the API key group and original requested model. Reuse the same family defaults as `OpenAIGatewayHandler.Messages`.

**Step 2: Call ForwardAsAnthropic for OpenAI accounts**

In the existing forward branch:

```go
if account.Platform == service.PlatformAntigravity && account.Type != service.AccountTypeAPIKey {
    result, err = h.antigravityGatewayService.Forward(...)
} else if account.Platform == service.PlatformOpenAI {
    openAIResult, openAIErr := h.openAIGatewayService.ForwardAsAnthropic(...)
    result, err = convertOpenAIForwardResult(openAIResult), openAIErr
} else {
    result, err = h.gatewayService.Forward(...)
}
```

Use the original Anthropic body with channel model mapping applied when applicable.

**Step 3: Preserve failover behavior**

Ensure `UpstreamFailoverError` from the OpenAI path is handled by the existing failover block. Do not fail over after streaming bytes are written.

**Step 4: Nil dependency guard**

If an OpenAI account is selected but `openAIGatewayService` is nil, return a service-unavailable Anthropic error instead of panicking.

### Task 3: Add tests

**Files:**
- Modify/Create: `backend/internal/handler/gateway_handler_openai_dispatch_test.go`

**Step 1: Test dispatch helper or branch**

Prefer extracting a small helper if direct full-handler testing is too heavy. Test that selected `PlatformOpenAI` routes to the OpenAI compatibility forwarder.

**Step 2: Test mapped model**

Verify Claude model family requests pass the resolved OpenAI mapped model (`gpt-5.3-codex` for Sonnet by default, or exact configured mapping if present).

**Step 3: Regression test non-OpenAI branch**

Verify a non-OpenAI account still uses the existing native forward path.

**Step 4: Run tests**

Run:

```bash
go test ./internal/handler ./internal/service -run 'Test.*OpenAI.*Messages|TestForwardAsAnthropic|TestResolveOpenAIMessagesDispatchMappedModel' -count=1
```

Expected: all targeted tests pass.

### Task 4: Build image tar

**Files:**
- Use: `deploy/scripts/build-image-tar.sh`

**Step 1: Run build**

Run from repo root:

```bash
deploy/scripts/build-image-tar.sh
```

If BuildKit hits the known Go compiler resource crash, use the established low-concurrency/legacy builder path instead of silently retrying.

**Step 2: Verify artifact**

Confirm `sub2api-<commit>.tar.gz` and `.sha256` exist under `/data/tmp/sub2api-images`.

### Task 5: Deploy to xrouter and verify

**Files:**
- Use: `deploy/scripts/deploy-image-tar.sh`
- Use: `/data/workspace/openclaw/main/direct_ssh.sh`

**Step 1: Upload artifact**

Copy tar and sha256 to `root@47.236.19.206:/opt/sub2api/images`.

**Step 2: Deploy**

Run remote deploy script with the new image tag.

**Step 3: Health check**

Run:

```bash
curl -fsS http://127.0.0.1:18080/health
cat /opt/sub2api/scripts/current-successful-release
docker compose -f docker-compose.local.yml -f docker-compose.vps.yml ps sub2api
```

Expected: health returns `{"status":"ok"}` and container is healthy.

**Step 4: Live API verification**

Send a minimal Anthropic `/v1/messages` request to `https://xrouter.uk/v1/messages` with the test Claude key and model `gpt-5.5`.

Expected: not `503 no available accounts`, not `401 Invalid bearer token`, not `502`; response is Anthropic-compatible.

# Platformless Groups Capability Routing Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Make groups platformless so any group can expose any supported API protocol when its account pool contains a capable account.

**Architecture:** Replace route-time `group.platform` dispatch with endpoint/protocol dispatch. Add protocol-aware account selection that lists group accounts without platform filtering, filters by account capability and model support, then forwards through the native handler or existing compatibility converter based on selected account platform/type.

**Tech Stack:** Go 1.26.3, Gin, PostgreSQL/Ent, Redis, existing `GatewayService`, `OpenAIGatewayService`, Gemini/Antigravity compatibility services, Docker deployment scripts.

---

## Context

Design doc: `docs/plans/2026-05-12-platformless-groups-capability-routing-design.md`

Current branch: `fix/copilot-model-selection`

Important existing files:

- `backend/internal/server/routes/gateway.go`
- `backend/internal/handler/gateway_handler.go`
- `backend/internal/handler/gateway_handler_chat_completions.go`
- `backend/internal/handler/gateway_handler_responses.go`
- `backend/internal/handler/openai_gateway_handler.go`
- `backend/internal/handler/gemini_v1beta_handler.go`
- `backend/internal/service/gateway_service.go`
- `backend/internal/service/openai_gateway_service.go`
- `backend/internal/service/gateway_forward_as_chat_completions.go`
- `backend/internal/service/gateway_forward_as_responses.go`
- `backend/internal/service/openai_gateway_messages.go`
- `backend/internal/service/openai_account_scheduler.go`
- `backend/internal/handler/endpoint.go`

Important existing tests:

- `backend/internal/server/routes/gateway_test.go`
- `backend/internal/handler/openai_gateway_handler_test.go`
- `backend/internal/handler/gemini_v1beta_handler_test.go`
- `backend/internal/handler/gateway_handler_stream_failover_test.go`
- `backend/internal/service/gateway_group_isolation_test.go`
- `backend/internal/service/openai_compat_model_test.go`
- `backend/internal/service/api_key_service_cache_test.go`

Use `/usr/local/go/bin/go` for local tests because `/usr/bin/go` is Go 1.18 and cannot parse `go 1.26.3`.

---

## Task 1: Add protocol/capability primitives

**Files:**

- Create: `backend/internal/service/inbound_protocol.go`
- Test: `backend/internal/service/inbound_protocol_test.go`

**Step 1: Write failing tests**

Create `backend/internal/service/inbound_protocol_test.go` with table tests:

```go
package service

import "testing"

func TestAccountSupportsInboundProtocol(t *testing.T) {
	tests := []struct {
		name     string
		account  Account
		protocol InboundProtocol
		want     bool
	}{
		{"anthropic_native_messages", Account{Platform: PlatformAnthropic}, InboundProtocolAnthropicMessages, true},
		{"openai_messages_compat", Account{Platform: PlatformOpenAI}, InboundProtocolAnthropicMessages, true},
		{"gemini_messages_compat", Account{Platform: PlatformGemini}, InboundProtocolAnthropicMessages, true},
		{"antigravity_messages", Account{Platform: PlatformAntigravity}, InboundProtocolAnthropicMessages, true},
		{"openai_native_chat", Account{Platform: PlatformOpenAI}, InboundProtocolOpenAIChatCompletions, true},
		{"anthropic_chat_compat", Account{Platform: PlatformAnthropic}, InboundProtocolOpenAIChatCompletions, true},
		{"openai_native_responses", Account{Platform: PlatformOpenAI}, InboundProtocolOpenAIResponses, true},
		{"anthropic_responses_compat", Account{Platform: PlatformAnthropic}, InboundProtocolOpenAIResponses, true},
		{"openai_images", Account{Platform: PlatformOpenAI}, InboundProtocolOpenAIImagesGenerations, true},
		{"anthropic_no_images", Account{Platform: PlatformAnthropic}, InboundProtocolOpenAIImagesGenerations, false},
		{"gemini_native", Account{Platform: PlatformGemini}, InboundProtocolGeminiV1Beta, true},
		{"openai_no_gemini", Account{Platform: PlatformOpenAI}, InboundProtocolGeminiV1Beta, false},
		{"codex_openai_oauth", Account{Platform: PlatformOpenAI, Type: AccountTypeOAuth}, InboundProtocolCodexResponses, true},
		{"codex_openai_apikey_false", Account{Platform: PlatformOpenAI, Type: AccountTypeAPIKey}, InboundProtocolCodexResponses, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := AccountSupportsInboundProtocol(&tt.account, tt.protocol); got != tt.want {
				t.Fatalf("AccountSupportsInboundProtocol() = %v, want %v", got, tt.want)
			}
		})
	}
}
```

**Step 2: Run test to verify it fails**

Run:

```bash
cd backend
/usr/local/go/bin/go test ./internal/service -run TestAccountSupportsInboundProtocol -count=1
```

Expected: fail because `InboundProtocol` and `AccountSupportsInboundProtocol` are undefined.

**Step 3: Implement primitives**

Create `backend/internal/service/inbound_protocol.go`:

```go
package service

// InboundProtocol identifies the client-facing API protocol for capability-aware scheduling.
type InboundProtocol string

const (
	InboundProtocolAnthropicMessages        InboundProtocol = "anthropic_messages"
	InboundProtocolAnthropicCountTokens     InboundProtocol = "anthropic_count_tokens"
	InboundProtocolOpenAIChatCompletions    InboundProtocol = "openai_chat_completions"
	InboundProtocolOpenAIResponses          InboundProtocol = "openai_responses"
	InboundProtocolOpenAIImagesGenerations  InboundProtocol = "openai_images_generations"
	InboundProtocolOpenAIImagesEdits        InboundProtocol = "openai_images_edits"
	InboundProtocolGeminiV1Beta             InboundProtocol = "gemini_v1beta"
	InboundProtocolCodexResponses           InboundProtocol = "codex_responses"
)

func AccountSupportsInboundProtocol(account *Account, protocol InboundProtocol) bool {
	if account == nil {
		return false
	}
	switch protocol {
	case InboundProtocolAnthropicMessages:
		switch account.Platform {
		case PlatformAnthropic, PlatformOpenAI, PlatformGemini, PlatformAntigravity:
			return true
		default:
			return false
		}
	case InboundProtocolAnthropicCountTokens:
		return account.Platform == PlatformAnthropic
	case InboundProtocolOpenAIChatCompletions:
		return account.Platform == PlatformOpenAI || account.Platform == PlatformCopilot || account.Platform == PlatformAnthropic
	case InboundProtocolOpenAIResponses:
		return account.Platform == PlatformOpenAI || account.Platform == PlatformAnthropic
	case InboundProtocolOpenAIImagesGenerations, InboundProtocolOpenAIImagesEdits:
		return account.Platform == PlatformOpenAI
	case InboundProtocolGeminiV1Beta:
		return account.Platform == PlatformGemini
	case InboundProtocolCodexResponses:
		return (account.Platform == PlatformOpenAI && account.Type == AccountTypeOAuth) || account.Platform == PlatformCopilot
	default:
		return false
	}
}
```

If `PlatformCopilot` does not exist or uses another constant, inspect `backend/internal/service/account.go` and adjust the test and implementation to the real constant.

**Step 4: Run test to verify it passes**

Run:

```bash
cd backend
/usr/local/go/bin/go test ./internal/service -run TestAccountSupportsInboundProtocol -count=1
```

Expected: PASS.

**Step 5: Commit**

```bash
git add backend/internal/service/inbound_protocol.go backend/internal/service/inbound_protocol_test.go
git commit -m "feat: add inbound protocol account capabilities"
```

---

## Task 2: Add capability-aware scheduler selection

**Files:**

- Modify: `backend/internal/service/gateway_service.go`
- Test: create `backend/internal/service/gateway_protocol_selection_test.go`

**Step 1: Write failing scheduler tests**

Create `backend/internal/service/gateway_protocol_selection_test.go`. Reuse existing stubs from `gateway_group_isolation_test.go` if available. Add tests that construct a `GatewayService` with account repo stubs returning mixed accounts.

Test scenarios:

1. `SelectAccountForProtocolWithLoadAwareness` with `InboundProtocolAnthropicMessages` can select OpenAI account from group even when group platform is arbitrary/empty.
2. `InboundProtocolOpenAIImagesGenerations` skips Anthropic account and selects OpenAI account.
3. `InboundProtocolGeminiV1Beta` returns no available accounts when group only contains OpenAI accounts.
4. Force platform context still limits selection to forced platform.

Expected skeleton:

```go
func TestSelectAccountForProtocolWithLoadAwarenessFiltersByCapability(t *testing.T) {
	groupID := int64(10)
	svc := newGatewayServiceWithAccountsForTest([]Account{
		{ID: 1, Platform: PlatformAnthropic, Status: StatusActive, Schedulable: true, Concurrency: 1, AccountGroups: []AccountGroup{{GroupID: groupID}}},
		{ID: 2, Platform: PlatformOpenAI, Status: StatusActive, Schedulable: true, Concurrency: 1, AccountGroups: []AccountGroup{{GroupID: groupID}}},
	})

	got, err := svc.SelectAccountForProtocolWithLoadAwareness(context.Background(), &groupID, InboundProtocolOpenAIImagesGenerations, "", "gpt-image-1", nil, "", 0)
	require.NoError(t, err)
	require.Equal(t, int64(2), got.Account.ID)
}
```

Adjust helper names to match existing test stubs.

**Step 2: Run test to verify it fails**

```bash
cd backend
/usr/local/go/bin/go test ./internal/service -run TestSelectAccountForProtocol -count=1
```

Expected: fail because the new method does not exist.

**Step 3: Implement method**

In `backend/internal/service/gateway_service.go`, add:

```go
func (s *GatewayService) SelectAccountForProtocolWithLoadAwareness(
	ctx context.Context,
	groupID *int64,
	protocol InboundProtocol,
	sessionHash string,
	requestedModel string,
	excludedIDs map[int64]struct{},
	metadataUserID string,
	sub2apiUserID int64,
) (*AccountSelectionResult, error) {
	ctx = WithInboundProtocol(ctx, protocol) // only if Task 2 adds context helper; otherwise omit.
	return s.selectAccountWithLoadAwarenessFiltered(ctx, groupID, sessionHash, requestedModel, excludedIDs, metadataUserID, sub2apiUserID, func(account *Account) bool {
		return AccountSupportsInboundProtocol(account, protocol)
	})
}
```

Refactor existing `SelectAccountWithLoadAwareness` minimally:

- Extract its body into a private helper that accepts an optional `accountFilter func(*Account) bool`.
- Apply the filter after `listSchedulableAccountsForModel` and before building `accountByID`.
- Also apply the filter in the non-load-batch path after `SelectAccountForModelWithExclusions`; if the selected account fails the filter, add it to exclusions and continue.
- Do not change force-platform behavior.

Keep `SelectAccountWithLoadAwareness` as a wrapper with `nil` filter to reduce churn.

**Step 4: Run tests**

```bash
cd backend
/usr/local/go/bin/go test ./internal/service -run 'TestSelectAccountForProtocol|TestAccountSupportsInboundProtocol' -count=1
```

Expected: PASS.

**Step 5: Commit**

```bash
git add backend/internal/service/gateway_service.go backend/internal/service/gateway_protocol_selection_test.go
git commit -m "feat: select accounts by inbound protocol capability"
```

---

## Task 3: Remove route-time group platform dispatch

**Files:**

- Modify: `backend/internal/server/routes/gateway.go`
- Test: modify `backend/internal/server/routes/gateway_test.go`

**Step 1: Write failing route tests**

Update `gateway_test.go` so group platform no longer controls handler choice.

Target assertions:

- `/v1/messages` always calls `h.Gateway.Messages` or a new unified handler, not `h.OpenAIGateway.Messages` based on group platform.
- `/v1/chat/completions` always calls the unified Chat handler.
- `/v1/responses` always calls the unified Responses handler.
- `/v1/images/generations` no longer 404s solely because group platform is not OpenAI.

If existing route tests use fake handlers, add counters for each route target.

**Step 2: Run tests to verify failure**

```bash
cd backend
/usr/local/go/bin/go test ./internal/server/routes -run TestGatewayRoutes -count=1
```

Expected: fail because current routes branch on `getGroupPlatform`.

**Step 3: Modify routes**

In `backend/internal/server/routes/gateway.go`:

- Change `/v1/messages` to `h.Gateway.Messages`.
- Change `/v1/messages/count_tokens` to `h.Gateway.CountTokens` for now.
- Change `/v1/responses` and aliases to `h.Gateway.Responses` unless Task 5 creates a new unified handler wrapper.
- Change `/v1/chat/completions` and aliases to `h.Gateway.ChatCompletions` unless Task 5 creates a wrapper.
- Change `/v1/images/generations` and `/v1/images/edits` to `h.OpenAIGateway.Images` for now, but remove group platform guard.
- Keep `/v1beta` Gemini routes as-is at route level.
- Keep `/antigravity/*` force-platform routes as-is.
- Delete `getGroupPlatform` if unused.

**Step 4: Run route tests**

```bash
cd backend
/usr/local/go/bin/go test ./internal/server/routes -count=1
```

Expected: PASS.

**Step 5: Commit**

```bash
git add backend/internal/server/routes/gateway.go backend/internal/server/routes/gateway_test.go
git commit -m "fix: route gateway endpoints without group platform"
```

---

## Task 4: Make `/v1/messages` fully protocol-capability based

**Files:**

- Modify: `backend/internal/handler/gateway_handler.go`
- Modify: `backend/internal/handler/openai_gateway_handler.go` only if shared helper extraction is needed
- Test: create or modify `backend/internal/handler/gateway_handler_openai_dispatch_test.go`

**Step 1: Write failing tests**

Add tests covering:

1. OpenAI group with `AllowMessagesDispatch=false` can use `/v1/messages` when selected account is OpenAI.
2. Anthropic group with only OpenAI account can use `/v1/messages`.
3. Mixed group with Gemini account can use Gemini compat path if existing `GatewayHandler.Messages` already has Gemini handling.
4. No capable account returns Anthropic-formatted 503 with capability message.

Use existing `TestResolveOpenAIMessagesDispatchMappedModel` patterns and handler stubs where possible.

**Step 2: Run tests to verify failure**

```bash
cd backend
/usr/local/go/bin/go test ./internal/handler -run 'TestGatewayMessages.*Protocol|TestGatewayMessages.*OpenAI' -count=1
```

Expected: at least one test fails due to old selection API or missing capability error.

**Step 3: Update handler selection**

In `GatewayHandler.Messages`:

- Replace `SelectAccountWithLoadAwareness(...)` with `SelectAccountForProtocolWithLoadAwareness(..., service.InboundProtocolAnthropicMessages, ...)`.
- Remove reliance on `apiKey.Group.Platform` for Gemini sticky path where possible. If a Gemini-specific sticky hash is still required, derive it from selected protocol/account capability, not group platform.
- Keep selected-account forwarding logic:
  - OpenAI -> `h.openAIGatewayService.ForwardAsAnthropic`
  - Antigravity -> `h.antigravityGatewayService.Forward`
  - else -> `h.gatewayService.Forward`
- Do not check `AllowMessagesDispatch` anywhere in this path.

**Step 4: Improve capability error**

When selection fails with no candidates, return Anthropic error body like:

```json
{
  "type": "error",
  "error": {
    "type": "api_error",
    "message": "No available accounts: group has no schedulable account capable of anthropic_messages"
  }
}
```

Keep existing error status conventions unless tests expect a specific code.

**Step 5: Run targeted tests**

```bash
cd backend
/usr/local/go/bin/go test ./internal/handler -run 'TestGatewayMessages.*Protocol|TestGatewayMessages.*OpenAI|TestResolveOpenAIMessagesDispatchMappedModel' -count=1
```

Expected: PASS.

**Step 6: Commit**

```bash
git add backend/internal/handler/gateway_handler.go backend/internal/handler/gateway_handler_openai_dispatch_test.go
git commit -m "fix: dispatch messages by account capability"
```

---

## Task 5: Make OpenAI Chat and Responses handlers account-capability based

**Files:**

- Modify: `backend/internal/handler/gateway_handler_chat_completions.go`
- Modify: `backend/internal/handler/gateway_handler_responses.go`
- Modify: `backend/internal/service/openai_gateway_service.go` if a reusable native-forward helper is needed
- Test: create `backend/internal/handler/gateway_handler_openai_protocol_test.go`

**Step 1: Write failing tests**

Tests:

1. `/v1/chat/completions` with a group containing an OpenAI account uses native OpenAI Chat path or existing OpenAI gateway path, not Anthropic conversion.
2. `/v1/chat/completions` with a group containing only Anthropic account uses `GatewayService.ForwardAsChatCompletions`.
3. `/v1/responses` with a group containing OpenAI account uses native `OpenAIGatewayService` Responses path.
4. `/v1/responses` with a group containing only Anthropic account uses `GatewayService.ForwardAsResponses`.

If the current OpenAI native methods are handler-only and hard to call after pre-selected account, first write tests around behavior using fake upstreams and selected account stubs.

**Step 2: Run tests to verify failure**

```bash
cd backend
/usr/local/go/bin/go test ./internal/handler -run 'TestGateway(ChatCompletions|Responses).*Capability' -count=1
```

Expected: fail because current `GatewayHandler.ChatCompletions` and `Responses` always use Anthropic conversion after selection.

**Step 3: Implement selection change**

In `GatewayHandler.ChatCompletions`:

- Use `SelectAccountForProtocolWithLoadAwareness(..., service.InboundProtocolOpenAIChatCompletions, ...)`.
- If selected account is OpenAI/Copilot and native OpenAI Chat path exists, forward natively.
- Else if selected account is Anthropic, keep `ForwardAsChatCompletions`.
- Else return capability error or failover.

In `GatewayHandler.Responses`:

- Use `SelectAccountForProtocolWithLoadAwareness(..., service.InboundProtocolOpenAIResponses, ...)`.
- If selected account is OpenAI, forward through existing OpenAI Responses native path.
- Else if selected account is Anthropic, keep `ForwardAsResponses`.
- Else return capability error or failover.

Prefer extracting service-level native forward helpers from `OpenAIGatewayHandler` if needed, rather than calling a handler that reselects account.

**Step 4: Run targeted tests**

```bash
cd backend
/usr/local/go/bin/go test ./internal/handler ./internal/service -run 'TestGateway(ChatCompletions|Responses).*Capability|TestForwardAs(ChatCompletions|Responses)' -count=1
```

Expected: PASS.

**Step 5: Commit**

```bash
git add backend/internal/handler/gateway_handler_chat_completions.go backend/internal/handler/gateway_handler_responses.go backend/internal/handler/gateway_handler_openai_protocol_test.go backend/internal/service/openai_gateway_service.go
git commit -m "fix: dispatch openai protocols by account capability"
```

---

## Task 6: Make images and Gemini endpoints capability based

**Files:**

- Modify: `backend/internal/server/routes/gateway.go` if not finished in Task 3
- Modify: `backend/internal/handler/openai_gateway_handler.go`
- Modify: `backend/internal/handler/gemini_v1beta_handler.go`
- Test: modify `backend/internal/handler/gemini_v1beta_handler_test.go`
- Test: add image capability tests if existing image tests are not enough

**Step 1: Write failing tests**

Image tests:

- Group platform Anthropic with OpenAI image-capable account can call `/v1/images/generations`.
- Group with only Anthropic accounts returns OpenAI-formatted capability error, not route-level 404.

Gemini tests:

- Group platform OpenAI with Gemini account can call `/v1beta/models`.
- Group with no Gemini account returns Google-formatted capability error.

**Step 2: Run tests to verify failure**

```bash
cd backend
/usr/local/go/bin/go test ./internal/handler -run 'Test.*(Images|Gemini).*Capability' -count=1
```

Expected: fail because handlers currently check group platform or routes reject by platform.

**Step 3: Implement images capability dispatch**

- Ensure image routes no longer check `group.platform`.
- In image handler, select with:
  - `InboundProtocolOpenAIImagesGenerations` for generations
  - `InboundProtocolOpenAIImagesEdits` for edits
- Only OpenAI-capable accounts should pass.
- Return OpenAI error schema when none are capable.

**Step 4: Implement Gemini capability dispatch**

In `GeminiV1Beta*` handlers:

- Remove checks requiring `apiKey.Group.Platform == service.PlatformGemini`.
- Select with `InboundProtocolGeminiV1Beta`.
- Existing ForcePlatform middleware for `/antigravity/v1beta` remains authoritative.
- Return Google error schema when no Gemini-capable account exists.

**Step 5: Run targeted tests**

```bash
cd backend
/usr/local/go/bin/go test ./internal/handler -run 'Test.*(Images|Gemini).*Capability' -count=1
```

Expected: PASS.

**Step 6: Commit**

```bash
git add backend/internal/server/routes/gateway.go backend/internal/handler/openai_gateway_handler.go backend/internal/handler/gemini_v1beta_handler.go backend/internal/handler/gemini_v1beta_handler_test.go
git commit -m "fix: dispatch image and gemini endpoints by capability"
```

---

## Task 7: Remove `allow_messages_dispatch` from runtime gating

**Files:**

- Modify: `backend/internal/handler/openai_gateway_handler.go`
- Modify: `backend/internal/service/openai_messages_dispatch.go` if it force-disables field
- Test: modify `backend/internal/handler/openai_gateway_handler_test.go`
- Test: modify `backend/internal/service/api_key_service_cache_test.go` only if snapshot expectations need updating

**Step 1: Write/adjust failing tests**

Add a test proving `AllowMessagesDispatch=false` does not block `/v1/messages` when a capable account exists.

**Step 2: Run test to verify failure**

```bash
cd backend
/usr/local/go/bin/go test ./internal/handler -run 'Test.*AllowMessagesDispatch' -count=1
```

Expected: fail if any runtime gate remains.

**Step 3: Remove runtime checks**

- Remove this block from `OpenAIGatewayHandler.Messages`:

```go
if apiKey.Group != nil && !apiKey.Group.AllowMessagesDispatch { ... }
```

- Do not remove fields from DTOs/DB in this pass.
- If `openai_messages_dispatch.go` intentionally forces `AllowMessagesDispatch=false` for non-OpenAI groups, leave it only if it is admin/UI normalization and no longer affects runtime. Otherwise remove or neutralize it.

**Step 4: Run tests**

```bash
cd backend
/usr/local/go/bin/go test ./internal/handler ./internal/service -run 'Test.*AllowMessagesDispatch|TestAPIKey.*Cache' -count=1
```

Expected: PASS.

**Step 5: Commit**

```bash
git add backend/internal/handler/openai_gateway_handler.go backend/internal/service/openai_messages_dispatch.go backend/internal/handler/openai_gateway_handler_test.go backend/internal/service/api_key_service_cache_test.go
git commit -m "fix: stop gating messages dispatch by group flag"
```

---

## Task 8: Replace model routing checks that depend on group platform

**Files:**

- Modify: `backend/internal/service/gateway_service.go`
- Test: modify or add `backend/internal/service/gateway_model_routing_platformless_test.go`

**Step 1: Write failing tests**

Test that model routing rules apply even when `group.Platform` is not Anthropic or is empty.

Example:

- Group has `ModelRoutingEnabled=true` and rule for `gpt-5.5` pointing to account ID 2.
- Group platform is `openai` or empty.
- Protocol is `anthropic_messages` or `openai_chat_completions`.
- Selection prefers account ID 2 if it is capable.

**Step 2: Run test to verify failure**

```bash
cd backend
/usr/local/go/bin/go test ./internal/service -run TestPlatformlessModelRouting -count=1
```

Expected: fail because current code gates routing with `group.Platform == PlatformAnthropic`.

**Step 3: Implement**

In `SelectAccountWithLoadAwareness` extracted helper:

- Change:

```go
if group != nil && requestedModel != "" && group.Platform == PlatformAnthropic { ... }
```

- To:

```go
if group != nil && requestedModel != "" { ... }
```

- Ensure capability filter still runs against routing candidates, so model routing cannot force an incapable account.

**Step 4: Run tests**

```bash
cd backend
/usr/local/go/bin/go test ./internal/service -run 'TestPlatformlessModelRouting|TestSelectAccountForProtocol' -count=1
```

Expected: PASS.

**Step 5: Commit**

```bash
git add backend/internal/service/gateway_service.go backend/internal/service/gateway_model_routing_platformless_test.go
git commit -m "fix: make model routing platformless"
```

---

## Task 9: Full test pass and static checks

**Files:** none expected unless tests expose bugs.

**Step 1: Run targeted suites**

```bash
cd backend
/usr/local/go/bin/go test ./internal/server/routes ./internal/handler ./internal/service -count=1
```

Expected: PASS.

**Step 2: Run broader backend tests if feasible**

```bash
cd backend
/usr/local/go/bin/go test ./internal/... -count=1
```

Expected: PASS or report unrelated pre-existing failures with exact test names. Do not hide failures.

**Step 3: Run git diff check**

```bash
git diff --check
```

Expected: no output.

**Step 4: Commit fixes if any**

```bash
git add <fixed files>
git commit -m "test: cover platformless gateway routing"
```

---

## Task 10: Build and deploy xrouter

**Files:** deployment scripts only used, not modified unless broken.

**Step 1: Check working tree**

```bash
git status --short
```

Expected: clean.

**Step 2: Build and publish image tar**

```bash
COMMIT=$(git rev-parse --short HEAD)
REPO_DIR=/data/workspace/sub2api-copilot-model-selection \
OUTPUT_DIR=/data/tmp/sub2api-images \
IMAGE_TAG="yelog/sub2api:${COMMIT}-v0.1.125" \
deploy/scripts/publish-image-tar.sh
```

Expected:

- Docker image builds.
- Tar uploads to `root@47.236.19.206:/opt/sub2api/images`.
- Remote `docker load` succeeds.
- Compose starts `sub2api`.
- Health check succeeds.

If Docker build fails due to known Go compiler resource crash, use the existing low-concurrency/legacy builder workaround documented in repo/deploy notes. Do not silently retry without explaining.

**Step 3: Verify remote health**

```bash
/data/workspace/openclaw/main/direct_ssh.sh root@47.236.19.206 'set -e; curl -fsS http://127.0.0.1:18080/health; echo; cat /opt/sub2api/scripts/current-successful-release; cd /opt/sub2api/deploy && docker compose -f docker-compose.local.yml -f docker-compose.vps.yml ps sub2api'
```

Expected:

- `{"status":"ok"}`
- state file references new image tag
- container status healthy

---

## Task 11: Online protocol verification

**Files:** none.

**Step 1: Verify `/v1/messages` with `test-claude`**

Use the known test key only for necessary verification. Do not print full keys in logs or final answer.

Expected: `HTTP/2 200`, body text `ok`.

**Step 2: Verify `/v1/messages` with `ymbp`**

Fetch key from remote DB by name or use known local secure source. Request:

```json
{"model":"gpt-5.5","max_tokens":16,"messages":[{"role":"user","content":"Reply with only: ok"}]}
```

Expected: `HTTP/2 200`, body text `ok`, even if `allow_messages_dispatch=false` and regardless of `groups.platform`.

**Step 3: Verify `/v1/chat/completions`**

Request with `ymbp`:

```json
{"model":"gpt-5.5","messages":[{"role":"user","content":"Reply with only: ok"}],"max_tokens":16}
```

Expected: `HTTP/2 200`, OpenAI Chat Completions-shaped response.

**Step 4: Verify `/v1/responses` or `/backend-api/codex/responses`**

Request with `ymbp`:

```json
{"model":"gpt-5.5","input":"Reply with only: ok"}
```

Expected: `HTTP/2 200`, OpenAI Responses-shaped response.

**Step 5: Check logs for old failure signatures**

```bash
/data/workspace/openclaw/main/direct_ssh.sh root@47.236.19.206 'cd /opt/sub2api/deploy && docker compose -f docker-compose.local.yml -f docker-compose.vps.yml logs --since=10m sub2api | grep -Ei "This group does not allow /v1/messages dispatch|no available accounts|Invalid bearer token|gateway.forward_failed| 502 | 503 " | tail -120 || true'
```

Expected: no relevant failures from verification requests.

---

## Final report

Report:

- Commit range implemented.
- Tests run and exact pass/fail status.
- Deployed image tag.
- Health check result.
- Online verification results for `test-claude`, `ymbp`, `/v1/messages`, `/v1/chat/completions`, `/v1/responses` or Codex path.
- Any remaining limitations, especially unsupported protocol/account combinations.

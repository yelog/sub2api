# Cross-Platform Model Routing Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Allow grouped requests with an explicit model to select any account in the group that supports that model, regardless of the group's legacy/default platform.

**Architecture:** Add a narrow cross-platform candidate path in `GatewayService` for grouped, non-forced, model-specific scheduling. Reuse existing account model capability checks and downstream scheduling constraints instead of adding model prefix routing.

**Tech Stack:** Go, existing GatewayService scheduling tests.

---

### Task 1: Add Cross-Platform Candidate Listing

**Files:**
- Modify: `backend/internal/service/account_service.go`
- Modify: `backend/internal/service/gateway_service.go`
- Test: `backend/internal/service/gateway_account_selection_test.go` or closest existing scheduling test file

**Step 1: Write the failing test**

Add a scheduler unit test where:

- Group ID is set.
- Group platform is `anthropic`.
- Group contains an OpenAI account with `model_mapping` exposing `gpt-5.5` or otherwise supporting `gpt-5.5`.
- Request model is `gpt-5.5`.
- `SelectAccountForModelWithExclusions` or `SelectAccountWithLoadAwareness` returns the OpenAI account.

**Step 2: Run test to verify failure**

Run from `backend`: `go test ./internal/service -run Test.*CrossPlatform.*Model -count=1`

Expected: FAIL with no available accounts or selected account not OpenAI.

**Step 3: Add repository method if needed**

If `AccountRepository` lacks a non-platform grouped schedulable list, reuse existing `ListSchedulableByGroupID(ctx, groupID)` method already present in the interface.

**Step 4: Implement helper**

In `GatewayService`, add a helper like:

```go
func (s *GatewayService) shouldUseCrossPlatformModelRouting(groupID *int64, requestedModel string, hasForcePlatform bool) bool {
    return groupID != nil && strings.TrimSpace(requestedModel) != "" && !hasForcePlatform
}
```

**Step 5: Use helper in account listing**

Before platform-filtered `listSchedulableAccounts` for the relevant scheduling path, list all schedulable group accounts when the helper returns true.

Keep forced platform and empty model behavior unchanged.

**Step 6: Run targeted test**

Run from `backend`: `go test ./internal/service -run Test.*CrossPlatform.*Model -count=1`

Expected: PASS.

### Task 2: Preserve Forced Platform Behavior

**Files:**
- Test: `backend/internal/service/gateway_account_selection_test.go` or closest existing scheduling test file
- Modify: `backend/internal/service/gateway_service.go` if test fails

**Step 1: Write forced platform test**

Create a test where context has `ctxkey.ForcePlatform = PlatformAntigravity`, group contains both OpenAI and Antigravity accounts, requested model is `gpt-5.5`, and only Antigravity candidates are considered.

**Step 2: Run targeted test**

Run from `backend`: `go test ./internal/service -run Test.*ForcePlatform.* -count=1`

Expected: PASS. If it fails, adjust helper usage so `hasForcePlatform` disables cross-platform listing.

### Task 3: Preserve Empty Model Behavior

**Files:**
- Test: `backend/internal/service/gateway_account_selection_test.go` or closest existing scheduling test file
- Modify: `backend/internal/service/gateway_service.go` if test fails

**Step 1: Write empty model test**

Create a test where group platform is `anthropic`, group contains Anthropic and OpenAI accounts, requested model is empty, and the selected account remains platform-compatible with legacy behavior.

**Step 2: Run targeted test**

Run from `backend`: `go test ./internal/service -run Test.*EmptyModel.* -count=1`

Expected: PASS.

### Task 4: Targeted Handler Smoke Test

**Files:**
- Test: existing handler test if lightweight; otherwise skip and rely on service coverage.

**Step 1: Add handler-level test only if existing stubs make it cheap**

If there is an existing `/v1/messages` handler test with fake gateway service, add one assertion that `gpt-5.5` is passed into scheduling and no platform-prefix decision is made in the handler.

**Step 2: Run targeted handler tests**

Run from `backend`: `go test ./internal/handler -run Test.*Messages.* -count=1`

Expected: PASS or skip if unrelated macOS path test makes package-level handler tests noisy.

### Task 5: Verification

**Files:**
- No planned code changes.

**Step 1: Run service tests**

Run from `backend`: `go test ./internal/service -count=1`

Expected: PASS.

**Step 2: Run frontend unaffected checks only if frontend changed**

No frontend change expected. Skip unless implementation touches UI.

**Step 3: Document known unrelated failures**

If full backend tests still fail on `TestResolvePageImagePath` due macOS `/private/var`, record it as unrelated to this change.

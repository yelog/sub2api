# Copilot Account Test Token Fix Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Fix GitHub Copilot OAuth account connection tests so they use Copilot credentials (`copilot_token` / `github_access_token`) instead of Claude OAuth `access_token`.

**Architecture:** Add a Copilot-specific account test path before the Claude fallback. The path applies existing model mapping, reads `copilot_token`, refreshes/exchanges it from `github_access_token` when missing or expired, and sends a Copilot-compatible request through the existing upstream HTTP abstraction.

**Tech Stack:** Go 1.26.3 backend, Gin SSE test endpoint, existing `internal/pkg/copilot` helpers, existing account repository update flow, Vitest unaffected.

---

### Task 1: Reproduce the real credential shape in tests

**Files:**
- Modify: `backend/internal/service/account_test_service_copilot_test.go`

**Steps:**
1. Change the existing Copilot account test fixture to use `copilot_token` instead of fake `access_token`.
2. Run targeted Go test in Docker if resources allow:
   `docker run --rm -v "$PWD/backend":/src -w /src golang:1.26.3-alpine go test ./internal/service -run TestAccountTestService_TestAccountConnection_CopilotAppliesModelMapping -count=1`
3. Expected before implementation: FAIL with `No access token available`.

### Task 2: Add Copilot-specific test connection path

**Files:**
- Modify: `backend/internal/service/account_test_service.go`

**Steps:**
1. Route `PlatformCopilot` before Claude fallback.
2. Add `testCopilotAccountConnection`.
3. Apply `account.GetMappedModel(modelID)`.
4. Resolve token:
   - use non-expired `credentials.copilot_token` first;
   - if missing/expired, exchange `credentials.github_access_token` via `copilot.ExchangeToken`;
   - persist refreshed token fields when repository supports update.
5. Send the same minimal Claude-compatible test payload with Authorization `Bearer <copilot_token>`.
6. Preserve SSE events and error behavior.

### Task 3: Add missing-token/refresh tests

**Files:**
- Modify: `backend/internal/service/account_test_service_copilot_test.go`

**Steps:**
1. Add test for real credentials using only `copilot_token` + `github_access_token`.
2. Add test ensuring missing both tokens returns a Copilot-specific error.
3. If practical, add exchange-from-GitHub-token test with a local HTTP server.

### Task 4: Validate, commit, deploy

**Steps:**
1. Run targeted backend tests in Docker if resource permits.
2. Run affected frontend tests if unchanged only as smoke: `./node_modules/.bin/vitest run src/components/account/__tests__/AccountTestModal.spec.ts`.
3. Commit fix.
4. Build image with explicit `VERSION=0.1.126` and new commit tag.
5. Transfer tar to VPS, `docker load`, deploy.
6. Verify container healthy and version/commit logs.

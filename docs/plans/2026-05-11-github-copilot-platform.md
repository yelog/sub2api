# GitHub Copilot Platform Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add GitHub Copilot as an independent `copilot` platform in admin account management, including OAuth account creation, reauth, and Premium requests usage display as `used / limit (percentage)`.

**Architecture:** Extend the existing admin account platform system with a dedicated `copilot` branch across frontend enums/UI and backend account/usage services. Reuse existing OAuth modal shells and account list plumbing, but keep Copilot credentials, usage parsing, and display semantics independent from OpenAI.

**Tech Stack:** Vue 3, TypeScript, Pinia, vue-i18n, Go, Gin, Ent repository layer, existing admin account services/tests.

---

### Task 1: Add failing frontend type and platform presentation tests for `copilot`

**Files:**
- Modify: `frontend/src/types/index.ts`
- Modify: `frontend/src/utils/platformColors.ts`
- Modify: `frontend/src/components/common/PlatformTypeBadge.vue`
- Test: `frontend/src/components/common/__tests__/PlatformTypeBadge.spec.ts`

**Step 1: Write the failing test**

Cover:
- `platformLabel('copilot') === 'GitHub Copilot'`
- `PlatformTypeBadge` renders Copilot label/class without falling back to default

**Step 2: Run test to verify it fails**

Run: `pnpm vitest frontend/src/components/common/__tests__/PlatformTypeBadge.spec.ts`
Expected: FAIL because `copilot` is not recognized.

**Step 3: Write minimal implementation**

- Add `'copilot'` to `AccountPlatform`
- Add Copilot color/label mapping
- Update badge component platform label/class branches

**Step 4: Run test to verify it passes**

Run: `pnpm vitest frontend/src/components/common/__tests__/PlatformTypeBadge.spec.ts`
Expected: PASS

**Step 5: Commit**

```bash
git add frontend/src/types/index.ts frontend/src/utils/platformColors.ts frontend/src/components/common/PlatformTypeBadge.vue frontend/src/components/common/__tests__/PlatformTypeBadge.spec.ts
git commit -m "feat: add copilot platform presentation"
```

### Task 2: Add failing admin create-account modal tests for Copilot platform visibility and flow selection

**Files:**
- Modify: `frontend/src/components/account/CreateAccountModal.vue`
- Modify: `frontend/src/components/account/OAuthAuthorizationFlow.vue`
- Test: `frontend/src/components/account/__tests__/CreateAccountModal.spec.ts`

**Step 1: Write the failing test**

Cover:
- Add Account modal shows GitHub Copilot platform option
- Selecting Copilot switches form into OAuth-only branch
- Copilot flow does not expose OpenAI-only manual token methods

**Step 2: Run test to verify it fails**

Run: `pnpm vitest frontend/src/components/account/__tests__/CreateAccountModal.spec.ts`
Expected: FAIL because Copilot platform option does not exist.

**Step 3: Write minimal implementation**

- Add Copilot to the modal platform selection UI
- Ensure account category/type resolves to `oauth`
- Pass `platform="copilot"` into OAuth flow
- Hide OpenAI/Gemini/Antigravity-specific manual input methods for Copilot

**Step 4: Run test to verify it passes**

Run: `pnpm vitest frontend/src/components/account/__tests__/CreateAccountModal.spec.ts`
Expected: PASS

**Step 5: Commit**

```bash
git add frontend/src/components/account/CreateAccountModal.vue frontend/src/components/account/OAuthAuthorizationFlow.vue frontend/src/components/account/__tests__/CreateAccountModal.spec.ts
git commit -m "feat: expose copilot in account creation flow"
```

### Task 3: Add failing frontend Copilot OAuth API/composable tests

**Files:**
- Create: `frontend/src/api/admin/copilot.ts`
- Create: `frontend/src/composables/useCopilotOAuth.ts`
- Test: `frontend/src/composables/__tests__/useCopilotOAuth.spec.ts`
- Modify: `frontend/src/api/admin/index.ts`

**Step 1: Write the failing test**

Cover:
- `startDeviceFlow` calls `/admin/copilot/oauth/start`
- `pollDeviceFlow` calls `/admin/copilot/oauth/poll`
- success response returns credentials and GitHub user info
- pending response remains pollable

**Step 2: Run test to verify it fails**

Run: `pnpm vitest frontend/src/composables/__tests__/useCopilotOAuth.spec.ts`
Expected: FAIL because Copilot composable/API do not exist.

**Step 3: Write minimal implementation**

- Add Copilot admin API wrapper
- Add composable for start/poll/reset/build-result state
- Wire into admin API index exports

**Step 4: Run test to verify it passes**

Run: `pnpm vitest frontend/src/composables/__tests__/useCopilotOAuth.spec.ts`
Expected: PASS

**Step 5: Commit**

```bash
git add frontend/src/api/admin/copilot.ts frontend/src/composables/useCopilotOAuth.ts frontend/src/composables/__tests__/useCopilotOAuth.spec.ts frontend/src/api/admin/index.ts
git commit -m "feat: add copilot oauth frontend client"
```

### Task 4: Add failing integration tests for creating Copilot OAuth accounts from admin UI payloads

**Files:**
- Modify: `frontend/src/components/account/CreateAccountModal.vue`
- Test: `frontend/src/components/account/__tests__/CreateAccountModal.spec.ts`

**Step 1: Write the failing test**

Cover:
- After Copilot OAuth completes, submit payload contains:
  - `platform: 'copilot'`
  - `type: 'oauth'`
  - Copilot credentials

**Step 2: Run test to verify it fails**

Run: `pnpm vitest frontend/src/components/account/__tests__/CreateAccountModal.spec.ts -t copilot`
Expected: FAIL because submit payload does not support Copilot.

**Step 3: Write minimal implementation**

- Add Copilot-specific success handling in modal
- Reuse existing account creation submit path
- Ensure validation and payload normalization allow `copilot`

**Step 4: Run test to verify it passes**

Run: `pnpm vitest frontend/src/components/account/__tests__/CreateAccountModal.spec.ts -t copilot`
Expected: PASS

**Step 5: Commit**

```bash
git add frontend/src/components/account/CreateAccountModal.vue frontend/src/components/account/__tests__/CreateAccountModal.spec.ts
git commit -m "feat: create copilot oauth accounts from admin modal"
```

### Task 5: Add failing backend tests for Copilot usage parsing and service behavior

**Files:**
- Create: `backend/internal/pkg/copilot/usage.go`
- Create: `backend/internal/pkg/copilot/usage_test.go`
- Create: `backend/internal/service/copilot_usage_service.go`
- Create: `backend/internal/service/copilot_usage_service_test.go`

**Step 1: Write the failing test**

Cover:
- Parse upstream usage payload into `used`, `limit`, `ratio`
- Handle missing fields gracefully
- Service marks expired/invalid token responses as actionable errors

**Step 2: Run test to verify it fails**

Run: `/usr/local/go/bin/go test ./internal/pkg/copilot ./internal/service -run 'TestCopilotUsage' -count=1`
Expected: FAIL because usage parser/service do not exist.

**Step 3: Write minimal implementation**

- Add Copilot usage response structs/parser
- Add service fetching Premium requests usage from upstream
- Normalize to a backend response DTO

**Step 4: Run test to verify it passes**

Run: `/usr/local/go/bin/go test ./internal/pkg/copilot ./internal/service -run 'TestCopilotUsage' -count=1`
Expected: PASS

**Step 5: Commit**

```bash
git add backend/internal/pkg/copilot/usage.go backend/internal/pkg/copilot/usage_test.go backend/internal/service/copilot_usage_service.go backend/internal/service/copilot_usage_service_test.go
git commit -m "feat: add copilot premium usage service"
```

### Task 6: Add failing backend handler/service tests for Copilot account usage API exposure

**Files:**
- Modify: `backend/internal/handler/admin`
- Modify: `backend/internal/service`
- Modify: `backend/internal/server/routes/admin.go`
- Test: relevant handler/service test files

**Step 1: Write the failing test**

Cover:
- Admin usage endpoint returns Copilot usage payload for Copilot OAuth account
- Non-Copilot account does not attempt Copilot usage fetch
- Errors are degraded/sanitized correctly

**Step 2: Run test to verify it fails**

Run: `/usr/local/go/bin/go test ./internal/handler/admin ./internal/service ./cmd/server -run 'Test.*Copilot.*Usage' -count=1`
Expected: FAIL because route/service wiring is missing.

**Step 3: Write minimal implementation**

- Inject Copilot usage service
- Wire route/handler or existing account-usage endpoint branch
- Return normalized `copilot_usage` payload

**Step 4: Run test to verify it passes**

Run: `/usr/local/go/bin/go test ./internal/handler/admin ./internal/service ./cmd/server -run 'Test.*Copilot.*Usage' -count=1`
Expected: PASS

**Step 5: Commit**

```bash
git add backend/internal/handler/admin backend/internal/service backend/internal/server/routes/admin.go backend/cmd/server/wire_gen.go backend/cmd/server/wire_gen_test.go
git commit -m "feat: expose copilot usage in admin APIs"
```

### Task 7: Add failing frontend tests for Copilot usage rendering in account list and reauth support

**Files:**
- Modify: `frontend/src/components/account/AccountUsageCell.vue`
- Modify: `frontend/src/components/admin/account/ReAuthAccountModal.vue`
- Test: `frontend/src/components/account/__tests__/AccountUsageCell.spec.ts`
- Test: `frontend/src/components/admin/account/__tests__/ReAuthAccountModal.spec.ts`

**Step 1: Write the failing test**

Cover:
- Copilot account usage cell displays `185 / 500 (37%)`
- Loading/error/empty states behave correctly
- Reauth modal routes Copilot account into Copilot OAuth flow

**Step 2: Run test to verify it fails**

Run: `pnpm vitest frontend/src/components/account/__tests__/AccountUsageCell.spec.ts frontend/src/components/admin/account/__tests__/ReAuthAccountModal.spec.ts`
Expected: FAIL because Copilot usage/reauth branches do not exist.

**Step 3: Write minimal implementation**

- Add Copilot usage rendering branch to `AccountUsageCell`
- Add Copilot branch to reauth modal
- Reuse shared OAuth UI where possible

**Step 4: Run test to verify it passes**

Run: `pnpm vitest frontend/src/components/account/__tests__/AccountUsageCell.spec.ts frontend/src/components/admin/account/__tests__/ReAuthAccountModal.spec.ts`
Expected: PASS

**Step 5: Commit**

```bash
git add frontend/src/components/account/AccountUsageCell.vue frontend/src/components/admin/account/ReAuthAccountModal.vue frontend/src/components/account/__tests__/AccountUsageCell.spec.ts frontend/src/components/admin/account/__tests__/ReAuthAccountModal.spec.ts
git commit -m "feat: render copilot usage and reauth flows"
```

### Task 8: Add/update i18n, docs, and end-to-end verification

**Files:**
- Modify: `frontend/src/i18n/locales/zh.ts`
- Modify: `frontend/src/i18n/locales/en.ts`
- Modify: docs as needed

**Step 1: Write the failing test**

If i18n snapshot/spec coverage exists, add/adjust it. Otherwise use direct verification in the UI/component tests above.

**Step 2: Run tests to verify failures or missing keys**

Run: `pnpm vitest`
Expected: FAIL on missing translations or Copilot branches.

**Step 3: Write minimal implementation**

- Add Copilot labels, OAuth strings, Premium requests strings
- Update any docs/comments that describe supported platforms

**Step 4: Run full targeted verification**

Run:
```bash
cd frontend && pnpm vitest
cd ../backend && /usr/local/go/bin/go test ./internal/pkg/copilot ./internal/service ./internal/handler/admin ./cmd/server -count=1
```
Expected: PASS

**Step 5: Commit**

```bash
git add frontend/src/i18n/locales/zh.ts frontend/src/i18n/locales/en.ts docs
git commit -m "feat: finalize copilot platform support"
```

### Task 9: Build, release, and deploy

**Files:**
- Modify: release artifacts only

**Step 1: Verify workspace diff**

Run: `git diff --check && git status --short`
Expected: clean formatting and intended file set only.

**Step 2: Build and package image**

Run the same local Docker build → save tar.gz → upload → `docker load` workflow already proven for this repo.

**Step 3: Deploy to VPS**

- Update `/opt/sub2api/deploy/.env` with new image tag
- `docker compose -f docker-compose.local.yml -f docker-compose.vps.yml up -d sub2api`

**Step 4: Validate deployment**

Run:
- container status check
- health check `curl http://127.0.0.1:18080/health`
- confirm running image tag
- smoke-check Copilot admin endpoints if safe

**Step 5: Commit / tag handoff**

```bash
git add -A
git commit -m "feat: add github copilot admin platform"
```

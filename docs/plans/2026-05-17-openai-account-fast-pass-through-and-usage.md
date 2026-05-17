# OpenAI Account Fast Pass-through and Usage Statistics Implementation Plan

> **For Hermes:** Use subagent-driven-development skill to implement this plan task-by-task.

**Goal:** Let admins decide per OpenAI account whether user-sent Codex `/fast` / `service_tier=fast|priority` is allowed through xrouter, defaulting to disabled, and make usage statistics distinguish fast vs non-fast traffic.

**Architecture:** Existing xrouter behavior already normalizes Codex `service_tier=fast` to `priority` and the default OpenAI fast policy strips `priority`, so `/fast` currently has no upstream effect. Keep the global policy as the safety baseline, then add an account-level override stored on the account config/extra and checked before global policy evaluation. Usage logs already persist `service_tier`; add aggregate views/filters that bucket `priority` as fast and everything else/null as non-fast.

**Tech Stack:** Go backend, Ent/Postgres migrations, Vue 3 + TypeScript frontend, existing admin account modal and usage views.

---

## Requirement Analysis

### Current behavior

- Codex `/fast` ultimately appears in request payload as `service_tier: "fast"`.
- Backend normalizes `fast` to OpenAI's upstream-recognized `priority`.
- `DefaultOpenAIFastPolicySettings()` is currently:
  - `service_tier = priority`
  - `action = filter`
  - `scope = all`
  - `model_whitelist = []` meaning all models
- `applyOpenAIFastPolicyToBody()` deletes `service_tier` when the rule resolves to `filter`.
- Therefore, with current default settings, user enabling Codex `/fast` through xrouter is effectively no-op for upstream priority routing: xrouter strips it before OpenAI.
- `usage_logs.service_tier` already exists and captures normalized tier in multiple paths, but current UI mostly shows per-log `service_tier`; it does not clearly summarize fast vs non-fast usage.

### Desired behavior

1. In **帐号管理** add/edit OpenAI account, provide a switch to enable/disable pass-through of user-sent `/fast`-type params.
2. Default is **off**.
3. Off means current behavior is preserved: `fast`/`priority` removed before upstream, so user `/fast` has no effect.
4. On means for that account, user `fast`/`priority` is preserved and sent upstream as `priority`.
5. Usage statistics can show fast and non-fast statistics separately.

### Proposed domain terms

- UI label: `允许用户 /fast 模式`
- Backend field: `openai_fast_passthrough_enabled` or `openai_allow_fast_passthrough`
- Fast bucket: `service_tier == "priority"` after normalization.
- Non-fast bucket: `service_tier IS NULL OR service_tier != "priority"`.

---

## UI / UE / UX Design

### 1. Account create/edit form

**Placement:** `帐号管理` → add/edit account modal → only visible when `platform === openai`.

**Control:** Toggle switch.

**Default:** Off.

**Label:** `允许用户 /fast 模式`

**Help text:**

- Off: `默认关闭。关闭时会移除 Codex /fast 传来的 service_tier=fast/priority，上游按普通优先级处理。`
- On: `开启后会透传为 OpenAI service_tier=priority，可能消耗 fast/priority 配额并影响成本/限额。`

**Microcopy:** Use warning styling only when on:

- badge: `可能消耗 fast 配额`
- tooltip: `/fast 是用户侧参数，和模型名无关；任何模型带 service_tier=fast/priority 都会走该开关。`

**Interactions:**

- Creating an OpenAI account starts off.
- Editing existing accounts with missing field displays off.
- Switching platform away from OpenAI hides the control and submits false/omits field.
- When on, account list/details may show a small badge: `Fast 透传`.

### 2. Account list / detail visibility

Add an optional badge near other account capability badges:

- Off: no badge, to reduce noise.
- On: badge `Fast 透传` with amber/blue color.

### 3. Usage statistics

Add fast/non-fast distinction in two layers:

1. **Usage table row:** existing `service_tier` display can label:
   - `priority` → `Fast`
   - null/empty → `Normal`
   - `flex` → `Flex`
   - `auto/default/scale` → original label
2. **Usage summary/cards:** add counters/costs:
   - `Fast 请求数`
   - `Fast Tokens`
   - `Fast 成本`
   - `非 Fast 请求数`
   - `非 Fast Tokens`
   - `非 Fast 成本`

**Filter UX:** Add dropdown `Fast 模式`:

- `全部`
- `Fast`
- `非 Fast`

**Export UX:** CSV includes `service_tier` and `fast_bucket` (`fast` / `non_fast`) for downstream analysis.

---

## Implementation Plan

### Task 1: Add account-level fast pass-through helper on backend

**Objective:** Define a single backend source of truth for reading/writing the per-account switch.

**Files:**
- Modify: `backend/internal/service/account.go`
- Test: `backend/internal/service/account_test.go` or new `backend/internal/service/openai_fast_policy_account_test.go`

**Steps:**
1. Add constant key: `openai_fast_passthrough_enabled`.
2. Add helper:
   - `func (a *Account) OpenAIFastPassthroughEnabled() bool`
   - returns false unless account platform is OpenAI and `Extra[key] == true`.
3. Add setter/normalizer helper if account creation/update code should sanitize non-OpenAI accounts.
4. Write tests:
   - nil account / nil extra → false
   - OpenAI with true → true
   - OpenAI with false/missing/string → false
   - non-OpenAI with true → false

**Verification:**

```bash
go test ./internal/service -run 'Test.*OpenAIFastPassthrough' -count=1
```

### Task 2: Wire create/update account DTOs and service persistence

**Objective:** Allow frontend to submit the switch without leaking it into credentials.

**Files:**
- Modify: `backend/internal/handler/admin/account_handler.go`
- Modify: `backend/internal/service/account_service.go` or `backend/internal/service/admin_service.go` depending on actual create/update path
- Modify: `frontend/src/types/index.ts`

**Steps:**
1. Add optional JSON field to create/update request:
   - `openai_fast_passthrough_enabled?: boolean`
2. In handler/service mapping, merge it into `Extra[openai_fast_passthrough_enabled]` only for OpenAI accounts.
3. Default missing value to false on create.
4. On update, distinguish missing from explicit false:
   - missing: leave existing value unchanged
   - false: set false/remove key
   - true: set true
5. Ensure bulk update either ignores this field or explicitly adds a later bulk-edit feature; do not accidentally change all accounts.

**Verification:**

```bash
go test ./internal/handler/admin ./internal/service -run 'Test.*Account.*Fast' -count=1
```

### Task 3: Apply account override before global fast policy

**Objective:** When the selected OpenAI account has pass-through enabled, preserve `service_tier=priority` instead of stripping it.

**Files:**
- Modify: `backend/internal/service/openai_gateway_service.go`
- Modify tests: `backend/internal/service/openai_fast_policy_test.go`
- Modify WS tests if WS frame policy path uses the same helper: `backend/internal/service/openai_fast_policy_ws_test.go`

**Steps:**
1. In `evaluateOpenAIFastPolicyWithSettings()` or a small wrapper before it, check:
   - account platform is OpenAI
   - normalized tier is `priority`
   - account fast pass-through is enabled
   - then return `pass`.
2. Keep `flex`, `auto`, `default`, `scale` governed by existing policy rules.
3. Keep block/filter behavior from explicit global admin policy only if deciding global policy should override account allow. Recommended precedence:
   - global `block` explicit should still block
   - account allow bypasses default filter only
   - implementation can mark default rule separately if needed. Simpler v1: account allow returns pass before settings; document that it overrides global filter for priority.
4. Ensure `fast` alias is still normalized to `priority` on pass.
5. Tests:
   - default account off strips `service_tier`.
   - account on preserves `"service_tier":"priority"`.
   - account on + raw `fast` rewrites to `priority`.
   - account on + `flex` still follows existing behavior.

**Verification:**

```bash
go test ./internal/service -run 'TestApplyOpenAIFastPolicyToBody|TestEvaluateOpenAIFastPolicy' -count=1
```

### Task 4: Update account UI form

**Objective:** Add a clear OpenAI-only switch in create/edit modal.

**Files:**
- Modify: `frontend/src/components/account/CreateAccountModal.vue`
- Modify edit account component if separate
- Modify translations under `frontend/src/locales/*` as applicable
- Modify: `frontend/src/types/index.ts`

**Steps:**
1. Extend form state with `openai_fast_passthrough_enabled: false`.
2. Show toggle only when platform is `openai`.
3. Add label/help/warning copy from UI design above.
4. Include the field in create/update payload for OpenAI only.
5. When loading an existing account, read from `account.extra.openai_fast_passthrough_enabled === true`.
6. Add optional account badge in list/detail component.

**Verification:**

```bash
cd frontend
pnpm typecheck
pnpm test -- --run account
```

Manual check:
- Add OpenAI account → toggle off by default.
- Edit existing OpenAI account → value persists.
- Non-OpenAI account → toggle hidden.

### Task 5: Add backend usage aggregation for fast/non-fast

**Objective:** Provide API-level statistics grouped by fast bucket.

**Files:**
- Inspect/modify: `backend/internal/repository/usage_log_repo*.go`
- Inspect/modify: `backend/internal/service/usage*.go`
- Inspect/modify: usage handler/routes under `backend/internal/handler/*usage*`
- Tests: usage repository/service tests

**Steps:**
1. Add optional filter param: `fast_bucket=fast|non_fast|all`.
2. Add aggregate fields to existing summary response or create a nested object:
   - `fast.request_count`, `fast.total_tokens`, `fast.total_cost`
   - `non_fast.request_count`, `non_fast.total_tokens`, `non_fast.total_cost`
3. SQL bucket definition:
   - fast: `service_tier = 'priority'`
   - non-fast: `service_tier IS NULL OR service_tier <> 'priority'`
4. Ensure indexes are adequate. Existing migration `070_add_usage_log_service_tier.sql` adds `(service_tier, created_at)`; likely sufficient.
5. Tests seed both priority and null/flex rows and verify counts/costs.

**Verification:**

```bash
go test ./internal/repository ./internal/service -run 'Test.*Usage.*Fast|Test.*ServiceTier' -count=1
```

### Task 6: Update usage UI summary/filter/export

**Objective:** Make fast vs non-fast visible to admins/users.

**Files:**
- Modify: `frontend/src/views/admin/UsageView.vue`
- Modify: `frontend/src/views/user/UsageView.vue`
- Modify: `frontend/src/components/admin/usage/UsageTable.vue`
- Modify: `frontend/src/types/index.ts`
- Modify API adapters under `frontend/src/api/*usage*`

**Steps:**
1. Add `fast_bucket` filter state and bind to query params.
2. Add summary cards/chips for fast and non-fast.
3. Update service tier labels:
   - priority → Fast
   - empty → Normal
   - flex → Flex
4. Add export column `fast_bucket`.
5. Keep default view as `全部` to avoid surprising users.

**Verification:**

```bash
cd frontend
pnpm typecheck
pnpm test -- --run usage
```

Manual check:
- Filter Fast only shows priority logs.
- Filter Non-fast shows null/flex/other logs.
- Summary totals equal all logs when fast + non-fast are combined.

### Task 7: End-to-end validation

**Objective:** Prove the feature works across xrouter.

**Steps:**
1. Create/edit OpenAI account with switch off.
2. Send request through xrouter with `service_tier: "fast"`.
3. Verify upstream request body does not contain `service_tier`.
4. Enable switch.
5. Send same request.
6. Verify upstream request body contains `service_tier: "priority"`.
7. Verify usage log records `service_tier=priority` for pass-through requests and summary counts it as Fast.
8. Verify normal request without service tier counts as Non-fast.

**Commands:**

```bash
go test ./... -count=1
cd frontend && pnpm typecheck && pnpm build
```

---

## Open Questions / Decisions

1. **Precedence:** Should account-level allow override all global fast policy filters, or only the default filter? Recommendation for v1: account allow means pass-through for that account; global `block` should be reserved for system-wide emergency if technically easy.
2. **Storage:** Prefer reusing `account.extra` for this boolean to avoid migration. If the product wants query/filter by this field in account list, add a first-class column later.
3. **Stats scope:** Usage stats should initially group by recorded `service_tier`, not by account switch state, because the business question is “what traffic actually used fast”.

---

## Acceptance Criteria

- Existing OpenAI accounts default to fast pass-through off.
- New OpenAI account form defaults off.
- When off, Codex `/fast` has no upstream effect after xrouter.
- When on, Codex `/fast` reaches OpenAI as `service_tier=priority`.
- Usage UI can distinguish Fast (`priority`) and Non-fast traffic in summary and filtering.
- Existing global OpenAI fast policy UI continues working.
- Backend and frontend tests pass.

# Cross-Platform Groups Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Make group UX platform-neutral, allow cross-platform account assignment where the current stack supports it, and expand API key/CCS configuration flows for Claude Code, Codex CLI, OpenCode, Gemini CLI, Antigravity, Copilot CLI, and related clients.

**Architecture:** Phase 1 keeps the existing group data model and treats any stored group platform as legacy/default metadata. Frontend flows should no longer use group platform as the primary client selector for API key usage or CCS import. Backend changes are limited to removing platform equality validation if it exists on account-group assignment.

**Tech Stack:** Vue 3, TypeScript, vue-i18n, Vitest, Go, Gin/Ent.

---

### Task 1: CCS Import Client Types

**Files:**
- Modify: `frontend/src/utils/ccswitchImport.ts`
- Test: `frontend/src/utils/__tests__/ccswitchImport.spec.ts`

**Step 1: Write failing tests**

Add table tests covering target clients independent of group platform:

```ts
it.each([
  { clientType: 'claude', app: 'claude', endpointSuffix: '' },
  { clientType: 'codex', app: 'codex', endpointSuffix: '' },
  { clientType: 'gemini', app: 'gemini', endpointSuffix: '/v1beta' },
  { clientType: 'opencode', app: 'opencode', endpointSuffix: '' },
  { clientType: 'openclaw', app: 'openclaw', endpointSuffix: '' },
  { clientType: 'antigravity', app: 'claude', endpointSuffix: '/antigravity' },
  { clientType: 'copilot', app: 'copilot', endpointSuffix: '' }
])('builds CCS import config for $clientType', ({ clientType, app, endpointSuffix }) => {
  const url = buildCcSwitchImportDeeplink({
    baseUrl: 'https://api.example.com',
    clientType: clientType as CcSwitchClientType,
    providerName: 'test-key',
    apiKey: 'sk-test',
    usageScript: 'return true'
  })

  const params = new URL(url).searchParams
  expect(params.get('app')).toBe(app)
  expect(params.get('endpoint')).toBe(`https://api.example.com${endpointSuffix}`)
})
```

**Step 2: Run test to verify it fails**

Run: `pnpm test:run frontend/src/utils/__tests__/ccswitchImport.spec.ts`

Expected: FAIL because `CcSwitchClientType` only supports `claude | gemini` and config resolution is platform-driven.

**Step 3: Implement minimal code**

Change `CcSwitchClientType` to:

```ts
export type CcSwitchClientType = 'claude' | 'codex' | 'gemini' | 'opencode' | 'openclaw' | 'antigravity' | 'copilot'
```

Change `resolveCcSwitchImportConfig` to resolve primarily by `clientType`, not group platform. Keep `platform` as optional legacy input only for default behavior if needed.

**Step 4: Run test to verify it passes**

Run: `pnpm test:run frontend/src/utils/__tests__/ccswitchImport.spec.ts`

Expected: PASS.

### Task 2: CCS Import Selection Dialog

**Files:**
- Modify: `frontend/src/views/user/KeysView.vue`
- Modify: `frontend/src/i18n/locales/zh.ts`
- Modify: `frontend/src/i18n/locales/en.ts`

**Step 1: Write or update tests if an existing KeysView test exists**

Search for existing key view tests. If none exist, skip creating a large mount test and rely on utility tests plus build.

**Step 2: Update dialog options**

Replace the current two-button dialog with options for:

- Claude Code
- Codex CLI
- Gemini CLI
- OpenCode
- OpenClaw
- Antigravity
- Copilot CLI

Use a single `ccsClientOptions` array in script and render with `v-for` to avoid repeated markup.

**Step 3: Update handler type**

Change `handleCcsClientSelect` to accept the expanded `CcSwitchClientType` and pass it directly to `buildCcSwitchImportDeeplink`.

**Step 4: Update locale copy**

Update `keys.ccsClientSelect.description` to say the user must choose the target client before import. Add labels/descriptions for all new clients in both Chinese and English locale files.

**Step 5: Run targeted checks**

Run: `pnpm typecheck`

Expected: PASS.

### Task 3: API Key Usage Modal Cross-Platform Client Tabs

**Files:**
- Modify: `frontend/src/components/keys/UseKeyModal.vue`
- Modify: `frontend/src/i18n/locales/zh.ts`
- Modify: `frontend/src/i18n/locales/en.ts`

**Step 1: Add client tabs**

Make `clientTabs` return the same high-level client set for any assigned group:

- Claude Code
- Codex CLI
- Codex CLI (WebSocket)
- OpenCode
- Gemini CLI
- Antigravity
- Copilot CLI

Keep the existing shell-tab behavior where applicable.

**Step 2: Add generators**

Reuse existing generators where possible:

- Claude Code: `generateAnthropicFiles(apiBase, apiKey)`
- Codex CLI: `generateOpenAIFiles(apiBase, apiKey)`
- Codex CLI WebSocket: `generateOpenAIWsFiles(apiBase, apiKey)`
- Gemini CLI: `generateGeminiCliContent(geminiBase, apiKey)`
- Antigravity: existing antigravity Claude/Gemini behavior or a dedicated tab showing both config blocks
- OpenCode: `generateOpenCodeConfig(...)`
- Copilot CLI: add a small config block using `GITHUB_COPILOT_API_KEY` or the project's existing Copilot CLI convention if already documented elsewhere

**Step 3: Update descriptions**

Add a neutral description explaining that model availability is determined by the accounts and permissions in the key's current group.

**Step 4: Avoid platform-gated tabs**

Do not hide Codex/Gemini/Copilot tabs just because the group has a stored `platform` value. The stored platform is legacy metadata in phase 1.

**Step 5: Run typecheck**

Run: `pnpm typecheck`

Expected: PASS.

### Task 4: Group Management Platform-Neutral Copy

**Files:**
- Modify: `frontend/src/views/admin/GroupsView.vue`
- Modify: `frontend/src/i18n/locales/zh.ts`
- Modify: `frontend/src/i18n/locales/en.ts`

**Step 1: Update visual treatment**

In the platform column, change the display from a single platform ownership badge to a legacy/default badge. Prefer copy such as `Default platform` or `Legacy platform` in English and `默认平台` or `兼容平台` in Chinese.

**Step 2: Update page/help copy**

Where group description/help text exists, describe groups as cross-platform account pools.

**Step 3: Keep filters if backend still supports them**

Do not remove the platform filter in phase 1 unless it breaks existing usage. Rename it to legacy/default platform filtering if copy exists.

**Step 4: Run typecheck**

Run: `pnpm typecheck`

Expected: PASS.

### Task 5: Account Group Assignment Validation

**Files:**
- Inspect: `frontend/src/views/admin/AccountsView.vue`
- Inspect: `frontend/src/components/admin/account/CreateAccountModal.vue`
- Inspect: `frontend/src/components/admin/account/EditAccountModal.vue`
- Inspect: `backend/internal/handler/*account*.go`
- Inspect: `backend/internal/service/*account*.go`

**Step 1: Search for platform-filtered groups**

Search for expressions like:

```ts
groups.filter((g) => g.platform === form.platform)
group.platform !== account.platform
```

and Go equivalents.

**Step 2: Write failing test if backend rejects cross-platform assignment**

If a backend validation rejects cross-platform group assignment, add a handler/service test proving an OpenAI account can be assigned to an Anthropic/Gemini legacy-platform group.

**Step 3: Remove frontend filtering**

Any account create/edit/bulk edit group dropdown should list all groups. Keep search and labels unchanged.

**Step 4: Relax backend validation if present**

Remove only the platform equality check. Keep existence, permission, and ownership checks.

**Step 5: Run tests**

Run frontend tests if touched components have tests. Run backend targeted tests for changed package.

Expected: PASS.

### Task 6: Full Verification

**Files:**
- No code changes unless fixing verification failures.

**Step 1: Frontend unit tests**

Run from `frontend`: `pnpm test:run`

Expected: PASS.

**Step 2: Frontend build**

Run from `frontend`: `pnpm build`

Expected: PASS.

**Step 3: Backend tests**

Run from `backend`: `go test ./...`

Expected: PASS.

### Task 7: VPS Release

**Files:**
- Inspect: `deploy/README.md`
- Inspect: `deploy/docker-compose.vps.yml`
- Inspect: existing deployment scripts in `deploy/`

**Step 1: Confirm deployment target**

Before deploying, confirm the VPS target, access method, and whether the current branch should be built locally or pulled on the VPS.

**Step 2: Deploy using existing workflow**

Use the repository's documented VPS deployment path. Do not invent a new deployment process.

**Step 3: Smoke test after release**

Verify:

- Admin group page loads.
- Account group selector shows all groups.
- API key usage modal shows all client tabs.
- CCS import asks for client type first.
- Existing gateway endpoints still respond.

Expected: all smoke checks pass.

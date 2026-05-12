# GitHub Copilot Model Selection Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add model whitelist/mapping selection for GitHub Copilot accounts in the add-account form.

**Architecture:** Reuse the existing frontend model restriction system. Add a curated Copilot model list to `useModelWhitelist.ts`, render the same whitelist/mapping UI for Copilot OAuth accounts in `CreateAccountModal.vue`, and persist selected models to `credentials.model_mapping` during Copilot OAuth completion.

**Tech Stack:** Vue 3, TypeScript, Vitest, Vue Test Utils, pnpm.

---

### Task 1: Add Copilot model list

**Files:**
- Modify: `frontend/src/composables/useModelWhitelist.ts`
- Test: `frontend/src/composables/__tests__/useModelWhitelist.spec.ts`

**Step 1: Write/extend test**

Create or extend a Vitest spec that asserts:

```ts
expect(getModelsByPlatform('copilot')).toEqual([
  'gpt-4.1',
  'gpt-4o',
  'gpt-4o-mini',
  'gpt-5',
  'gpt-5-mini',
  'gpt-5.1',
  'gpt-5.1-codex',
  'claude-sonnet-4',
  'claude-opus-4.1',
  'gemini-2.5-pro'
])
expect(getModelsByPlatform('copilot')).not.toContain('claude-sonnet-4-20250514')
```

**Step 2: Run failing test**

Run from `frontend`:

```bash
corepack pnpm vitest run src/composables/__tests__/useModelWhitelist.spec.ts
```

Expected: FAIL because Copilot currently falls back to Claude models.

**Step 3: Implement model list**

Add:

```ts
const copilotModels = [
  'gpt-4.1',
  'gpt-4o',
  'gpt-4o-mini',
  'gpt-5',
  'gpt-5-mini',
  'gpt-5.1',
  'gpt-5.1-codex',
  'claude-sonnet-4',
  'claude-opus-4.1',
  'gemini-2.5-pro'
]
```

Then add `case 'copilot': return copilotModels` in `getModelsByPlatform`.

**Step 4: Run test**

Expected: PASS.

---

### Task 2: Render Copilot model restriction UI

**Files:**
- Modify: `frontend/src/components/account/CreateAccountModal.vue`
- Test: `frontend/src/components/account/__tests__/CreateAccountModal.spec.ts` or create a focused Copilot spec.

**Step 1: Write/extend component test**

Assert that when the modal is open and the Copilot platform button is clicked, the model restriction label and Copilot models are available/rendered.

**Step 2: Implement UI**

Add a block similar to the existing OpenAI OAuth model restriction block:

```vue
<div
  v-if="form.platform === 'copilot' && accountCategory === 'oauth-based'"
  class="border-t border-gray-200 pt-4 dark:border-dark-600"
>
  ... existing whitelist/mapping controls ...
  <ModelWhitelistSelector v-model="allowedModels" :platform="form.platform" />
</div>
```

Reuse `modelRestrictionMode`, `allowedModels`, `modelMappings`, `presetMappings`, `addModelMapping`, `addPresetMapping`, and i18n keys already used by OpenAI OAuth.

**Step 3: Run component test**

Expected: PASS.

---

### Task 3: Persist Copilot model mapping

**Files:**
- Modify: `frontend/src/components/account/CreateAccountModal.vue`

**Step 1: Find Copilot OAuth completion**

Target function:

```ts
const handleExchangeCode = async () => {
  ...
  const result = await copilotOAuth.pollDeviceFlow(form.proxy_id)
  const credentials = copilotOAuth.buildCredentials(result)
  const extra = copilotOAuth.buildExtraInfo(result)
  await createAccountAndFinish('copilot', 'oauth', credentials, extra)
}
```

**Step 2: Add model mapping build**

Before `createAccountAndFinish`:

```ts
const modelMapping = buildModelMappingObject(modelRestrictionMode.value, allowedModels.value, modelMappings.value)
if (modelMapping) {
  credentials.model_mapping = modelMapping
}
```

**Step 3: Run tests**

Run targeted tests from `frontend`.

---

### Task 4: Quality gates and commit

**Files:**
- Modify: all files above
- Create: design and plan docs under `docs/plans/`

**Step 1: Clean generated pnpm workspace placeholder if needed**

```bash
rm -f frontend/pnpm-workspace.yaml
```

**Step 2: Run checks**

```bash
cd frontend
corepack pnpm vitest run src/composables/__tests__/useModelWhitelist.spec.ts
corepack pnpm vitest run src/components/account/__tests__/<copilot-test-file>.spec.ts
corepack pnpm vue-tsc --noEmit
corepack pnpm eslint .
cd ..
git diff --check
```

**Step 3: Commit**

```bash
git add docs/plans/2026-05-12-copilot-model-selection-design.md docs/plans/2026-05-12-copilot-model-selection.md frontend/src/composables/useModelWhitelist.ts frontend/src/composables/__tests__/useModelWhitelist.spec.ts frontend/src/components/account/CreateAccountModal.vue frontend/src/components/account/__tests__/<copilot-test-file>.spec.ts
git commit -m "feat: add copilot model selection"
```

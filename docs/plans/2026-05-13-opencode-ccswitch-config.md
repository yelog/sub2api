# OpenCode and CC-Switch Config Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Improve API key usage examples so OpenCode recognizes GPT image attachments and CC-Switch imports can target explicit app categories.

**Architecture:** Keep changes frontend-only. Enrich generated OpenCode model metadata in `UseKeyModal.vue`, and extend CC-Switch deeplink utilities plus the user API key dialog to carry an explicit `app` choice independent of group platform.

**Tech Stack:** Vue 3, TypeScript, Vitest, URLSearchParams-based deeplink generation.

---

### Task 1: Enrich OpenCode OpenAI model metadata

**Files:**
- Modify: `frontend/src/components/keys/UseKeyModal.vue`
- Test: `frontend/src/components/keys/__tests__/UseKeyModal.spec.ts`

**Steps:**
1. Add shared helpers for OpenCode OpenAI model capability metadata.
2. Apply helpers to GPT/Codex OpenAI models, especially `gpt-5.5`.
3. Add a test asserting `gpt-5.5` has `attachment: true`, image input modality, and `cost`.
4. Run `npm run test:run -- src/components/keys/__tests__/UseKeyModal.spec.ts`.

### Task 2: Add explicit CC-Switch app selection

**Files:**
- Modify: `frontend/src/utils/ccswitchImport.ts`
- Modify: `frontend/src/views/user/KeysView.vue`
- Modify: `frontend/src/i18n/locales/zh.ts`
- Modify: `frontend/src/i18n/locales/en.ts`
- Test: `frontend/src/utils/__tests__/ccswitchImport.spec.ts`

**Steps:**
1. Extend CC-Switch app type to include `claude`, `codex`, `gemini`, `opencode`, `openclaw`.
2. Make deeplink generation use selected app when provided.
3. Add app/category select to the import dialog.
4. Filter/prioritize model options by selected app family while falling back to all models.
5. Add tests for `opencode` and `openclaw` app params.
6. Run `npm run test:run -- src/utils/__tests__/ccswitchImport.spec.ts` and `npm run typecheck`.

### Task 3: Commit and publish

**Files:**
- Commit all source/test/plan files except `.learnings/`.

**Steps:**
1. Run frontend tests and typecheck.
2. Commit with `fix: improve opencode and ccswitch key configs`.
3. Build/publish image tar with existing deploy scripts.
4. Verify VPS health.

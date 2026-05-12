# Copilot Model Rates Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Show the provided GitHub Copilot model list and display model multipliers in the account model selector.

**Architecture:** Keep a static Copilot metadata table in `useModelWhitelist.ts`. Expose existing model ID arrays for persistence and a display metadata lookup for UI-only labels and multipliers. Render multipliers only for single-platform Copilot selectors.

**Tech Stack:** Vue 3, TypeScript, Vitest, Vue Test Utils.

---

### Task 1: Add Copilot model metadata

**Files:**
- Modify: `frontend/src/composables/useModelWhitelist.ts`
- Modify: `frontend/src/composables/__tests__/useModelWhitelist.spec.ts`

**Steps:**
1. Replace `copilotModels` with metadata-derived IDs from the screenshot.
2. Export `getModelDisplayMeta(platform, model)`.
3. Ensure `allModels` labels use Copilot labels where metadata exists.
4. Add tests for exact Copilot list and multiplier metadata.

### Task 2: Display multiplier badges in selector

**Files:**
- Modify: `frontend/src/components/account/ModelWhitelistSelector.vue`
- Create/Modify: `frontend/src/components/account/__tests__/ModelWhitelistSelector.spec.ts`

**Steps:**
1. Compute whether current selector is Copilot-only.
2. Attach Copilot display metadata to available options.
3. Render model label plus right-aligned multiplier badge for Copilot only.
4. Test Copilot shows `included` and premium multipliers.
5. Test non-Copilot selector does not show multiplier badges.

### Task 3: Run quality gates and commit

**Commands:**

```bash
cd /data/workspace/sub2api/frontend
./node_modules/.bin/vitest run src/composables/__tests__/useModelWhitelist.spec.ts src/components/account/__tests__/ModelWhitelistSelector.spec.ts src/components/account/__tests__/CreateAccountModal.spec.ts
./node_modules/.bin/vue-tsc --noEmit
./node_modules/.bin/eslint . --ext .vue,.js,.jsx,.cjs,.mjs,.ts,.tsx,.cts,.mts
cd /data/workspace/sub2api
git diff --check
git add docs/plans/2026-05-12-copilot-model-rates-design.md docs/plans/2026-05-12-copilot-model-rates.md frontend/src/composables/useModelWhitelist.ts frontend/src/composables/__tests__/useModelWhitelist.spec.ts frontend/src/components/account/ModelWhitelistSelector.vue frontend/src/components/account/__tests__/ModelWhitelistSelector.spec.ts
git commit -m "feat: show copilot model rates"
```

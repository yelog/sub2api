# Copilot Account Model Selection Fix Design

## Background

GitHub Copilot OAuth accounts can select models during creation, but the account edit modal does not show a model restriction section. The account test modal also lists only a few Claude models instead of the models selected for the Copilot account.

## Root Cause

The Copilot model flow is only partially wired:

- `CreateAccountModal.vue` renders a Copilot OAuth model restriction section and persists the selected models to `credentials.model_mapping`.
- `EditAccountModal.vue` renders an OAuth model restriction section only for OpenAI OAuth accounts, so Copilot OAuth accounts cannot view or update their saved mapping.
- `AccountHandler.GetAvailableModels` has branches for OpenAI, Gemini, Antigravity, and Anthropic. Copilot accounts fall through to the Anthropic OAuth branch and return `claude.DefaultModels`.
- Account testing posts the selected `model_id`, but Copilot currently follows the Claude test path. Mapping-mode tests must resolve the requested model through the account mapping before calling the upstream Copilot-compatible endpoint.

## Goals

- Treat Copilot as a first-class platform in the account model-selection flow.
- Preserve the existing creation behavior.
- Add edit-time model restriction UI and persistence for Copilot OAuth accounts.
- Make test model lists reflect the Copilot account's saved model mapping.
- Avoid changing unrelated OpenAI, Gemini, Anthropic, Antigravity, or Bedrock behavior.

## Proposed Approach

Use the same `credentials.model_mapping` convention already used by account creation:

- In the frontend edit modal, add a Copilot OAuth model restriction block using `ModelWhitelistSelector` with `platform="copilot"`.
- Reuse the existing whitelist/mapping state shape: `allowedModels`, `modelMappings`, and `modelRestrictionMode`.
- On edit modal initialization, load Copilot `credentials.model_mapping` exactly like OpenAI OAuth: all `from === to` means whitelist mode; otherwise mapping mode.
- On submit, persist Copilot OAuth `model_mapping` into credentials. Empty mapping means no restriction and removes the field.
- In the backend model-list endpoint, add a Copilot branch before Anthropic fallback. Return mapping keys when configured; otherwise return a curated Copilot default model list.
- In account testing, apply `account.GetMappedModel` for Copilot OAuth before sending the test request so mapping mode tests hit the actual upstream model.

## Backend Data Shape

Copilot model restrictions continue to use:

```json
{
  "credentials": {
    "model_mapping": {
      "requested-model": "actual-upstream-model"
    }
  }
}
```

Whitelist mode is represented as identity mappings:

```json
{
  "gpt-5.4": "gpt-5.4",
  "claude-sonnet-4.5": "claude-sonnet-4.5"
}
```

## Copilot Default Models

Add a backend Copilot default model list matching the frontend curated Copilot list in `useModelWhitelist.ts`. This ensures `/api/v1/admin/accounts/:id/models` can return Copilot models even when no mapping exists.

## Testing Plan

- Backend unit tests for `GetAvailableModels`:
  - Copilot OAuth with `model_mapping` returns mapping keys.
  - Copilot OAuth without `model_mapping` returns Copilot defaults and not Claude defaults.
- Backend unit test for account testing:
  - Copilot mapping mode resolves requested model to actual mapped model before test dispatch.
- Frontend unit tests:
  - `EditAccountModal` shows model restriction controls for Copilot OAuth.
  - It loads Copilot identity mapping as whitelist selections.
  - It saves updated Copilot model restrictions to `credentials.model_mapping`.
  - `AccountTestModal` continues to render models returned by the backend for Copilot accounts.

## Rollout / Risk

Risk is low because changes are scoped to `platform === 'copilot'` branches and reuse existing model-mapping conventions. The main compatibility concern is keeping backend Copilot defaults synchronized with the frontend list; tests should catch regressions around fallback behavior.

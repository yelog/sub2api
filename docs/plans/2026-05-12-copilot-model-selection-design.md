# GitHub Copilot Model Selection Design

## Problem

When creating an account and selecting `GitHub Copilot`, the form does not show model selection. Users cannot restrict the Copilot account to specific supported models.

Root causes:

- `frontend/src/components/account/CreateAccountModal.vue` has model restriction UI for API key, Bedrock, Antigravity, and OpenAI OAuth, but not for Copilot OAuth.
- `frontend/src/composables/useModelWhitelist.ts` has no Copilot model list and no `getModelsByPlatform('copilot')` branch.
- Without a Copilot branch, generic model helpers fall back to Claude models, which would be incorrect for Copilot.

## Goals

- Show a model whitelist selector after users choose `GitHub Copilot` in the add-account form.
- List curated GitHub Copilot supported models.
- Persist the selected models into `credentials.model_mapping` using the existing whitelist/mapping mechanism.
- Avoid changing backend APIs unless required.

## Non-goals

- Do not dynamically fetch GitHub Copilot `/models` during account creation. At this point the OAuth token may not exist yet.
- Do not redesign the model restriction component.
- Do not change existing OpenAI, Gemini, Anthropic, Antigravity, or Bedrock behavior.

## Approach

Use the existing frontend model restriction mechanism:

1. Add `copilotModels` in `frontend/src/composables/useModelWhitelist.ts`.
2. Return `copilotModels` from `getModelsByPlatform('copilot')`.
3. Add a Copilot OAuth model restriction block to `CreateAccountModal.vue`, parallel to the existing OpenAI OAuth block.
4. When completing Copilot OAuth account creation, build `credentials.model_mapping` from `modelRestrictionMode`, `allowedModels`, and `modelMappings`.
5. Add tests for the model helper and the add-account form behavior.

## Initial Copilot model list

Curated list:

- `gpt-4.1`
- `gpt-4o`
- `gpt-4o-mini`
- `gpt-5`
- `gpt-5-mini`
- `gpt-5.1`
- `gpt-5.1-codex`
- `claude-sonnet-4`
- `claude-opus-4.1`
- `gemini-2.5-pro`

This can be updated later if Copilot model support changes.

## Testing

- Add/extend unit tests for `useModelWhitelist.ts` to verify Copilot model list and no Claude fallback.
- Add/extend component test for `CreateAccountModal.vue` to verify Copilot renders model selection.
- Run targeted tests, typecheck, lint, and whitespace checks.

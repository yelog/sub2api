# Copilot Model Rates Design

## Goal

When creating a GitHub Copilot account, the model selector must show the Copilot model menu from the provided screenshot and display each model's cost multiplier.

## Requirements

- Replace the existing Copilot curated model list with the screenshot's models.
- Show user-facing model names and multipliers in the Copilot model dropdown.
- Preserve the saved model identifier format in `credentials.model_mapping`; multipliers are display-only.
- Do not change non-Copilot platform behavior.
- Keep model restriction and mapping behavior compatible with existing `buildModelMappingObject`.

## Model metadata

Use static curated metadata because the create-account flow may not have a Copilot token yet.

Premium models:

| ID | Label | Multiplier |
| --- | --- | --- |
| `claude-haiku-4.5` | Claude Haiku 4.5 | `0.33x` |
| `claude-opus-4.5` | Claude Opus 4.5 | `3x` |
| `claude-opus-4.6` | Claude Opus 4.6 | `3x` |
| `claude-opus-4.7` | Claude Opus 4.7 | `15x` |
| `claude-sonnet-4.5` | Claude Sonnet 4.5 | `1x` |
| `claude-sonnet-4.6` | Claude Sonnet 4.6 | `1x` |
| `gpt-5.2` | GPT-5.2 | `1x` |
| `gpt-5.2-codex` | GPT-5.2-Codex | `1x` |
| `gpt-5.3-codex` | GPT-5.3-Codex | `1x` |
| `gpt-5.4` | GPT-5.4 | `1x` |
| `gpt-5.4-mini` | GPT-5.4 mini | `0.33x` |
| `gpt-5.5` | GPT-5.5 | `7.5x` |
| `gemini-2.5-pro` | Gemini 2.5 Pro | `1x` |
| `gemini-3-flash-preview` | Gemini 3 Flash (Preview) | `0.33x` |
| `gemini-3.1-pro-preview` | Gemini 3.1 Pro (Preview) | `1x` |
| `grok-code-fast-1` | Grok Code Fast 1 | `0.25x` |

Standard models:

| ID | Label | Multiplier |
| --- | --- | --- |
| `gpt-4.1` | GPT-4.1 | `included` |
| `gpt-4o` | GPT-4o | `included` |
| `gpt-5-mini` | GPT-5 mini | `included` |

## Approach

1. Add Copilot metadata in `frontend/src/composables/useModelWhitelist.ts`.
2. Export a lookup helper for per-platform display metadata.
3. Update `allModels` so Copilot entries use screenshot labels.
4. Update `ModelWhitelistSelector.vue` to show multipliers only when the selected platform is exactly Copilot.
5. Update tests for the exact list and UI rendering.

## Testing

- `useModelWhitelist.spec.ts`: verify exact Copilot IDs and metadata.
- `ModelWhitelistSelector` tests: verify Copilot shows `included`/`15x`, and OpenAI does not show multiplier badges.
- Existing `CreateAccountModal` Copilot test remains valid.

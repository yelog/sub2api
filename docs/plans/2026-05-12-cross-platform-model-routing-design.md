# Cross-Platform Model Routing Design

## Background

After enabling cross-platform group assignment, Claude Code can import an API key for a group that contains OpenAI accounts and request `gpt-5.5` through the Anthropic Messages protocol. The current scheduler still resolves the scheduling platform from the group's legacy/default platform before model filtering. If the group platform is `anthropic`, OpenAI accounts are excluded before the scheduler checks whether any account supports `gpt-5.5`, causing `503 No available accounts`.

## Goal

For non-forced grouped requests with a concrete requested model, select accounts by model capability inside the current group instead of using the group's legacy platform as the first account filter.

## Non-Goals

- Do not infer platform from model prefixes.
- Do not remove or migrate the group `platform` field.
- Do not change explicit forced platform routes such as Antigravity routes.
- Do not change requests that have no requested model.

## Design

The scheduler should preserve existing platform-specific behavior except for grouped requests where all of these are true:

- There is a group ID.
- There is no `ForcePlatform` in context.
- The request has a non-empty model name.

For those requests, the scheduler should:

1. Load all schedulable accounts in the current group without filtering by platform.
2. Filter the candidate accounts with existing model capability logic, such as `IsSchedulableForModelWithContext`.
3. Continue applying existing scheduling constraints: group membership, sticky sessions, priority, account schedulability, model rate limits, quota, window cost, RPM, and concurrency.
4. Hydrate and return the selected account as before.

Forced platform routes should keep the current behavior. Requests without a model should keep legacy platform scheduling because the scheduler has no reliable model capability signal.

## Compatibility Notes

- This design works with account model mappings because the request model remains the external capability key. The upstream mapped model is still resolved later by existing account/platform forwarding logic.
- It avoids brittle model-prefix routing and supports custom model names as long as account capability metadata exposes those names.
- It should produce clearer behavior for cross-platform groups: the group is the account pool, while model availability determines eligible accounts.

## Risks

- If any account without explicit model restrictions is treated as supporting all models, cross-platform routing may over-select that account. Tests should verify current model capability semantics for OpenAI accounts.
- Handler forwarding must still support the selected account's platform. The immediate failure path is in account selection; if forwarding assumptions surface later, they should be fixed at the handler dispatch layer.

## Testing Plan

- Unit test scheduler selection with an `anthropic` legacy group containing an OpenAI account that supports `gpt-5.5`; request `gpt-5.5` and expect the OpenAI account.
- Unit test forced platform still filters by the forced platform.
- Unit test empty model still uses legacy group platform behavior.
- Run targeted backend tests for gateway scheduling.

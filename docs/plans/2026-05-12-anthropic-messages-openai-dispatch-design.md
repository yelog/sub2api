# Anthropic Messages to OpenAI Account Dispatch Design

## Context

Claude Code calls `POST /v1/messages` with Anthropic Messages payloads. The `default` group can contain OpenAI accounts so users can expose OpenAI-compatible models through Claude Code. Recent cross-platform model routing lets an Anthropic entrypoint select accounts from another platform, but the forward path still assumes the native Anthropic protocol unless the whole group platform is OpenAI.

Observed production failures:

- `503 No available accounts` when `/v1/messages` routing filters by the Anthropic path and cannot find a native Anthropic account.
- `502` after simplified requests select an OpenAI account but forward the Anthropic bearer/protocol to an OpenAI upstream, producing upstream `401 Invalid bearer token`.

## Goal

When `POST /v1/messages` selects an OpenAI account from an Anthropic/default group, forward through the existing OpenAI compatibility path so Anthropic Messages are converted to OpenAI Responses upstream and converted back to Anthropic Messages downstream.

## Non-goals

- Do not require changing the group platform to OpenAI.
- Do not replace existing Anthropic, Gemini, or Antigravity forwarding.
- Do not redesign scheduler behavior beyond the minimal selected-account platform dispatch.

## Approach Options

### Option A: Dispatch after account selection by selected account platform

After `GatewayHandler.Messages` selects an account, if `account.Platform == openai`, call `OpenAIGatewayService.ForwardAsAnthropic` with the original Anthropic request body and the effective mapped model. Keep existing Anthropic/Gemini/Antigravity paths unchanged.

Pros:
- Uses the scheduler's actual selected account.
- Reuses tested Anthropic → OpenAI Responses compatibility code.
- Preserves current group configuration and mixed routing model.

Cons:
- Requires wiring `OpenAIGatewayService` into `GatewayHandler`.
- Requires separate usage result mapping from `OpenAIForwardResult` to the normal usage path or a dedicated OpenAI usage record path.

### Option B: Route `/v1/messages` to OpenAI handler when a group contains OpenAI accounts

Change route dispatch before handler execution.

Pros:
- Smaller-looking route change.

Cons:
- Route layer lacks scheduler context.
- Can bypass mixed Anthropic/Gemini/Antigravity routing rules.
- Harder to reason about fallback and failover.

### Option C: Require users to mark the group platform as OpenAI

No code change.

Pros:
- Immediate workaround.

Cons:
- Does not support the intended default group mixed-account use case.
- User explicitly requested code-level fix.

## Selected Design

Use Option A.

`GatewayHandler.Messages` remains the entrypoint for Anthropic `/v1/messages` groups. It selects an account as it does today. Immediately before forwarding, it branches on `account.Platform`:

- `antigravity` non-API-key accounts: existing `AntigravityGatewayService.Forward`.
- `openai` accounts: new OpenAI compatibility branch using `OpenAIGatewayService.ForwardAsAnthropic`.
- all others: existing `GatewayService.Forward`.

For the OpenAI branch, derive a prompt cache key from the same session extraction logic used by `OpenAIGatewayHandler.Messages`. Use group messages-dispatch model mapping where applicable, and preserve channel model mapping behavior.

## Data Flow

1. Claude Code sends Anthropic Messages payload to `/v1/messages`.
2. API key auth resolves group and permissions.
3. `GatewayHandler.Messages` parses request and computes session hash.
4. Scheduler selects an account, possibly OpenAI due to cross-platform model routing.
5. If selected account is OpenAI:
   - normalize/resolve the effective OpenAI model;
   - call `OpenAIGatewayService.ForwardAsAnthropic`;
   - record usage through the OpenAI usage path or converted usage fields.
6. Client receives Anthropic-compatible response or Anthropic SSE stream.

## Error Handling

- OpenAI compatibility failover errors should participate in the existing failover loop where safe.
- If streaming has already written bytes, do not attempt failover.
- Non-failover errors should write an Anthropic-formatted error response if no response has been written.
- Preserve existing behavior for Anthropic/Gemini/Antigravity accounts.

## Tests

Add unit coverage for the platform dispatch decision:

- Anthropic `/v1/messages` path with selected OpenAI account uses `ForwardAsAnthropic`, not native Anthropic forward.
- Mapped model is passed as the OpenAI default mapped model for Claude model families.
- Existing non-OpenAI selected account behavior remains unchanged.

Run targeted Go tests for handler/service packages before building.

## Rollout

1. Commit this design.
2. Implement wiring and dispatch branch.
3. Add tests.
4. Build Docker image tar.
5. Deploy to xrouter VPS.
6. Verify `/health` and a live `/v1/messages` request using the test Claude key.

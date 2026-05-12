# Platformless Groups Capability Routing Design

**Status:** Approved by user on 2026-05-12  
**Branch:** `fix/copilot-model-selection`  
**Context:** xrouter / Sub2API gateway routing

## Problem

Sub2API still treats `groups.platform` as a routing primitive. That leaks an old assumption into user-facing behavior:

- `/v1/messages` chooses `GatewayHandler.Messages` or `OpenAIGatewayHandler.Messages` based on `group.platform`.
- `/v1/chat/completions`, `/v1/responses`, images, and Gemini routes also branch on `group.platform`.
- OpenAI groups additionally require `allow_messages_dispatch` for Anthropic Messages compatibility.
- Mixed account pools are only partially supported. Recent fixes allow an Anthropic group to select OpenAI accounts for `/v1/messages`, but that is still a patch on top of platform-bound routing.

The desired behavior is simpler: groups should not have a platform. A group is an API key policy and account pool. Any group should be able to serve any inbound protocol if one of its accounts can serve that protocol natively or through a supported converter.

## Goal

Deprecate group platform semantics and route requests by inbound protocol plus selected account capability. After this change, a group containing any mix of Anthropic, OpenAI, Gemini, Copilot, or Antigravity accounts can expose Claude Code, Codex, OpenAI Chat/Responses, Gemini-compatible, and other supported interfaces when the account pool has a capable account.

## Non-goals

- Do not remove the `groups.platform` database column in the first implementation pass.
- Do not redesign account credentials or OAuth flows.
- Do not build new protocol converters beyond the ones already available unless tests expose a small missing adapter needed for the target matrix.
- Do not make `/antigravity/*` force routes platformless; those routes intentionally force Antigravity accounts.

## Core model

### Group

A group is policy and membership only:

- API key grouping
- account pool membership
- quota / rate limit / pricing multiplier
- feature restrictions such as `claude_code_only`
- model routing and model mapping
- content moderation scope

`groups.platform` becomes legacy metadata. Request routing and account selection must not depend on it.

`allow_messages_dispatch` is deprecated. `/v1/messages` availability is determined by account capability, not a group flag.

### Inbound protocol

Add an explicit protocol concept, for example:

- `anthropic_messages`
- `anthropic_count_tokens`
- `openai_chat_completions`
- `openai_responses`
- `openai_images_generations`
- `openai_images_edits`
- `gemini_v1beta_models`
- `codex_responses`

Routes map URL and method to an inbound protocol. They do not inspect `group.platform`.

### Account capability

Capability is derived from account platform/type and available converters.

Initial matrix:

| Inbound protocol | Anthropic account | OpenAI account | Gemini account | Copilot account | Antigravity account |
|---|---:|---:|---:|---:|---:|
| `anthropic_messages` | native | via `ForwardAsAnthropic` | via Gemini compat | no, unless converter exists | via Antigravity handler |
| `openai_chat_completions` | via `ForwardAsChatCompletions` | native OpenAI gateway | maybe future | native/compatible if supported | no by default |
| `openai_responses` | via `ForwardAsResponses` | native OpenAI gateway | maybe future | maybe future | no by default |
| `openai_images_*` | no by default | native OpenAI image path | no by default | no by default | no by default |
| `gemini_v1beta_*` | no by default | no by default | native Gemini path | no by default | only force route if explicit Antigravity path |
| `codex_responses` | no by default | native OpenAI/Codex path | no by default | supported if existing path supports it | no by default |

The implementation can encode this matrix as functions first. A full table-driven capability registry can come later if needed.

## Routing design

### Route layer

The route layer should dispatch by endpoint only:

- `/v1/messages` -> unified Anthropic Messages handler
- `/v1/chat/completions` -> unified Chat Completions handler
- `/v1/responses` and aliases -> unified Responses handler
- `/v1/images/*` -> unified OpenAI Images handler
- `/v1beta/*` -> unified Gemini handler
- `/backend-api/codex/*` -> unified Codex/OpenAI responses handler

No route should call `getGroupPlatform(c)` for normal gateway dispatch.

### Handler layer

Each handler should:

1. Parse and validate the inbound request.
2. Identify inbound protocol.
3. Run group-level policy checks that are truly protocol-specific, such as `claude_code_only`.
4. Ask the scheduler for an account capable of the inbound protocol and requested model.
5. Forward via the correct native path or converter based on selected account platform/type.
6. Record usage through the matching usage recorder for the actual selected account path.

### Scheduler layer

Add capability-aware selection APIs, for example:

```go
type InboundProtocol string

const (
    InboundProtocolAnthropicMessages InboundProtocol = "anthropic_messages"
    InboundProtocolOpenAIChat        InboundProtocol = "openai_chat_completions"
    InboundProtocolOpenAIResponses   InboundProtocol = "openai_responses"
    InboundProtocolOpenAIImages      InboundProtocol = "openai_images"
    InboundProtocolGeminiV1Beta      InboundProtocol = "gemini_v1beta"
    InboundProtocolCodexResponses    InboundProtocol = "codex_responses"
)

func (s *GatewayService) SelectAccountForProtocolWithLoadAwareness(
    ctx context.Context,
    groupID *int64,
    protocol InboundProtocol,
    sessionHash string,
    requestedModel string,
    excludedIDs map[int64]struct{},
    metadataUserID string,
    sub2apiUserID int64,
) (*AccountSelectionResult, error)
```

This selection should list all schedulable accounts in the group, unless a force-platform context is present. Then it filters by:

- account schedulable status
- account belongs to group
- protocol capability
- model support/mapping
- routing rules
- quota/window/RPM/concurrency/session constraints

`force_platform` remains supported for explicit routes such as `/antigravity/*`.

## Error behavior

Errors should name the missing capability rather than blaming platform:

- No account supports the inbound protocol: `group has no schedulable account capable of anthropic_messages`
- Accounts support protocol but not model: `no schedulable account capable of anthropic_messages supports model gpt-5.5`
- Accounts exist but are limited: keep current quota/concurrency/failover messages where possible.

Protocol-specific error format must match the inbound protocol:

- Anthropic `/v1/messages` -> Anthropic error schema
- OpenAI Chat/Responses/images -> OpenAI error schema
- Gemini -> Google error schema

## Compatibility strategy

Phase 1 keeps database fields:

- `groups.platform` remains in DB and DTOs but routing ignores it.
- `allow_messages_dispatch` remains in DB and DTOs but handlers ignore it.
- Existing admin UI can still show fields until a later cleanup, but changing them should not affect gateway capability routing.

Phase 2 removes/hides UI fields and cleans DTOs.

Phase 3 migrates database columns if desired.

## Testing strategy

Add tests at three levels:

1. Route dispatch tests prove routes do not branch by `group.platform`.
2. Scheduler tests prove mixed groups are filtered by protocol capability.
3. Handler tests prove selected account platform determines native vs compatibility forwarder.

Minimum target cases:

- OpenAI group with `allow_messages_dispatch=false` can call `/v1/messages` using an OpenAI account.
- Anthropic group containing only OpenAI accounts can call `/v1/messages`.
- Mixed group uses Anthropic account natively for `/v1/messages` when available.
- Mixed group can use OpenAI account for `/v1/chat/completions` natively.
- Anthropic account can satisfy `/v1/chat/completions` through existing converter.
- Group with no protocol-capable accounts returns a clear protocol capability error.
- Explicit `/antigravity/*` routes still force Antigravity accounts.

## Risks

- Usage recording differs between `GatewayService` and `OpenAIGatewayService`. Each handler must record usage using the path matching the selected account and converter.
- Existing model routing currently gates some logic on `group.Platform == anthropic`. This needs to become platformless or protocol-aware.
- Sticky session keys may bind sessions to accounts that are not capable of a different inbound protocol. Capability filtering must run before sticky reuse succeeds.
- Some handlers currently parse requests into Anthropic-specific structures. Native OpenAI/Gemini dispatch paths should avoid unnecessary lossy conversion.

## Rollout

1. Implement behind tests on the existing branch.
2. Deploy to xrouter.
3. Verify with the known keys:
   - `test-claude` on `default`
   - `ymbp` on `VIP 满血版`
4. Verify protocols:
   - Claude Code / Anthropic `/v1/messages`
   - OpenAI `/v1/chat/completions`
   - OpenAI `/v1/responses`
   - Codex endpoint if available
   - Gemini `/v1beta/models/*` for Gemini-capable groups

## Success criteria

- No normal gateway route branches on `group.platform`.
- `allow_messages_dispatch` no longer blocks `/v1/messages`.
- Account selection considers all accounts in a group and filters by protocol capability.
- Online xrouter verification passes for Claude Code-style `/v1/messages` and OpenAI/Codex-style endpoints using groups that are not platform-specific.

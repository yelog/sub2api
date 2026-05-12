# Cross-Platform Groups Design

## Background

Groups currently appear platform-oriented in the admin and user UI. The target product model is different: a group should be an account pool that can contain accounts from any supported platform. API keys bound to that group should be able to call any supported client protocol, and scheduling should choose an eligible account in the current group based on model capability and account availability.

Supported client/protocol surfaces include Claude Code, Codex CLI, OpenCode, Gemini CLI, Antigravity, Copilot CLI, and future CCS-compatible clients such as OpenClaw.

## Goals

- Make group management platform-neutral from the user's perspective.
- Allow accounts from Claude, OpenAI, Gemini, Antigravity, Copilot, and future platforms to be assigned to the same group.
- Make API key usage documentation cover Claude Code, Codex CLI, OpenCode, Gemini CLI, Antigravity, and Copilot CLI.
- Change CCS import so the user selects the target client/platform before import configuration is generated.
- Preserve existing data compatibility and avoid a high-risk scheduling rewrite in the first phase.

## Non-Goals For Phase 1

- Do not remove existing group platform fields from the database.
- Do not rewrite the full scheduler in the same change unless a small backend validation block prevents cross-platform assignment.
- Do not guarantee every protocol can already route every model through every upstream platform. That belongs in the follow-up scheduler and protocol compatibility phase.

## Recommended Rollout

Use a phased compatibility rollout.

### Phase 1: Product Semantics And UX

- Update group UI copy to describe groups as cross-platform account pools.
- Stop filtering or blocking group selection by the account's platform in account management.
- Keep legacy platform metadata visible only where it still has operational meaning, and avoid presenting it as a hard ownership rule.
- Expand API key usage instructions to include all target clients.
- Make CCS import open a target-client selection dialog first, then generate import parameters for the selected client.
- If backend account-group assignment has a platform equality check, relax that check while preserving all existing fields.

### Phase 2: Unified Scheduling

- Treat a group as the scheduling boundary across all platforms.
- Resolve each request by API key, group, requested model, entry protocol, account capability, account health, quota, and concurrency.
- Return clear errors when the current group has no eligible account for the requested model/protocol combination.
- Add tests for cross-platform selection, model mapping, fallback behavior, and quota limits.

### Phase 3: Capability Matrix

- Add an explicit protocol/platform/model compatibility matrix.
- Use the matrix to drive UI hints, scheduling eligibility, and error messages.
- Document unsupported combinations instead of relying on implicit fallback behavior.

## UI / UX Design

### Group Management

- Rename explanatory copy from platform-specific grouping to cross-platform account pool wording.
- Where group cards or badges currently emphasize a single platform, use neutral labels such as `Cross-platform group` or `Account pool`.
- If existing groups still have a stored platform, show it as legacy/default metadata only when needed.

### Account Management

- The group selector should list all groups for all account platforms.
- The option rendering can still show group name, description, rates, and subscription type.
- Do not disable groups because their stored platform differs from the account platform.

### API Key Usage

The usage dialog should present client-specific cards or tabs for:

- Claude Code
- Codex CLI
- OpenCode
- Gemini CLI
- Antigravity
- Copilot CLI

Each client section should include the base URL, API key placement, and any client-specific configuration notes. The copy should clarify that available models depend on the accounts and model permissions in the key's current group.

### CCS Import

- Clicking `Import to CCS` first opens a target-client selection dialog.
- The target choices should include Claude Code, Codex CLI, Gemini CLI, OpenCode, OpenClaw, Antigravity, and Copilot CLI where supported by the existing importer.
- After selection, generate the import URL/config using the selected target client.
- Only ask for a model when the selected client import requires one.

## Data Compatibility

Keep the existing group data shape in phase 1. If a `platform` field exists on groups, do not delete or migrate it yet. Treat it as legacy/default metadata until phase 2 defines the final scheduling model.

## Testing Plan

- Frontend tests for account group selectors showing all groups regardless of account platform.
- Frontend tests for API key usage content including all target clients.
- Frontend tests for CCS import requiring target-client selection before generating import configuration.
- Backend tests only if backend validation currently rejects cross-platform account-group assignment.
- Build and test with frontend unit tests, frontend build, and backend `go test ./...` before release.

## Release Plan

- Implement phase 1 behind existing UI flows, without destructive migration.
- Verify locally with automated tests and targeted manual checks.
- Deploy to VPS using the repository's existing deployment workflow after tests pass.
- Monitor account assignment, API key usage, and gateway logs for unexpected scheduling errors.

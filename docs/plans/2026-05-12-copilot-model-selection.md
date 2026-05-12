# Copilot Model Selection Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Fix GitHub Copilot account model selection so create, edit, available-models, and account-test flows all use the saved Copilot model mapping.

**Architecture:** Copilot accounts continue to store model restrictions in `credentials.model_mapping`, using identity mappings for whitelist mode and non-identity mappings for mapping mode. The frontend edit modal gets a Copilot OAuth model restriction section and save/load wiring. The backend gets Copilot-specific default models, available-models response logic, and mapped-model resolution for account tests.

**Tech Stack:** Vue 3 + Vitest for frontend, Go + Gin + testify for backend, existing `service.Account.GetModelMapping/GetMappedModel` helpers.

---

### Task 1: Backend Copilot available models

**Files:**
- Modify: `backend/internal/handler/admin/account_handler.go`
- Modify/Test: `backend/internal/handler/admin/account_handler_available_models_test.go`
- Possibly create/modify: `backend/internal/pkg/copilot/models.go`

**Step 1: Write failing backend tests**

Add tests to `backend/internal/handler/admin/account_handler_available_models_test.go`:

```go
func TestAccountHandlerGetAvailableModels_CopilotOAuthUsesExplicitModelMapping(t *testing.T) {
    svc := &availableModelsAdminService{
        stubAdminService: newStubAdminService(),
        account: service.Account{
            ID:       44,
            Name:     "copilot-oauth",
            Platform: service.PlatformCopilot,
            Type:     service.AccountTypeOAuth,
            Status:   service.StatusActive,
            Credentials: map[string]any{
                "model_mapping": map[string]any{
                    "gpt-5.4": "gpt-5.4",
                    "claude-sonnet-4.5": "claude-sonnet-4.5",
                },
            },
        },
    }
    router := setupAvailableModelsRouter(svc)

    rec := httptest.NewRecorder()
    req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/accounts/44/models", nil)
    router.ServeHTTP(rec, req)

    require.Equal(t, http.StatusOK, rec.Code)

    var resp struct {
        Data []struct { ID string `json:"id"` } `json:"data"`
    }
    require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
    require.Len(t, resp.Data, 2)
    ids := []string{resp.Data[0].ID, resp.Data[1].ID}
    require.ElementsMatch(t, []string{"gpt-5.4", "claude-sonnet-4.5"}, ids)
}

func TestAccountHandlerGetAvailableModels_CopilotOAuthWithoutMappingUsesCopilotDefaults(t *testing.T) {
    svc := &availableModelsAdminService{
        stubAdminService: newStubAdminService(),
        account: service.Account{
            ID:       45,
            Name:     "copilot-oauth-default",
            Platform: service.PlatformCopilot,
            Type:     service.AccountTypeOAuth,
            Status:   service.StatusActive,
        },
    }
    router := setupAvailableModelsRouter(svc)

    rec := httptest.NewRecorder()
    req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/accounts/45/models", nil)
    router.ServeHTTP(rec, req)

    require.Equal(t, http.StatusOK, rec.Code)

    var resp struct {
        Data []struct { ID string `json:"id"` } `json:"data"`
    }
    require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
    require.NotEmpty(t, resp.Data)
    require.Contains(t, collectModelIDs(resp.Data), "gpt-5.4")
    require.NotEqual(t, "claude-opus-4-1-20250805", resp.Data[0].ID)
}
```

If no helper exists for IDs, add a small local helper in the test file.

**Step 2: Run failing tests**

Run:

```bash
go test ./backend/internal/handler/admin -run 'TestAccountHandlerGetAvailableModels_Copilot' -count=1
```

Expected: FAIL because Copilot falls through to Anthropic/Claude defaults.

**Step 3: Implement Copilot default models**

Create `backend/internal/pkg/copilot/models.go` with a simple model struct compatible with JSON response:

```go
package copilot

type Model struct {
    ID          string `json:"id"`
    Object      string `json:"object"`
    Type        string `json:"type"`
    DisplayName string `json:"display_name"`
}

var DefaultModels = []Model{
    {ID: "claude-haiku-4.5", Object: "model", Type: "model", DisplayName: "Claude Haiku 4.5"},
    {ID: "claude-opus-4.5", Object: "model", Type: "model", DisplayName: "Claude Opus 4.5"},
    {ID: "claude-opus-4.6", Object: "model", Type: "model", DisplayName: "Claude Opus 4.6"},
    {ID: "claude-opus-4.7", Object: "model", Type: "model", DisplayName: "Claude Opus 4.7"},
    {ID: "claude-sonnet-4.5", Object: "model", Type: "model", DisplayName: "Claude Sonnet 4.5"},
    {ID: "claude-sonnet-4.6", Object: "model", Type: "model", DisplayName: "Claude Sonnet 4.6"},
    {ID: "gpt-5.2", Object: "model", Type: "model", DisplayName: "GPT-5.2"},
    {ID: "gpt-5.2-codex", Object: "model", Type: "model", DisplayName: "GPT-5.2-Codex"},
    {ID: "gpt-5.3-codex", Object: "model", Type: "model", DisplayName: "GPT-5.3-Codex"},
    {ID: "gpt-5.4", Object: "model", Type: "model", DisplayName: "GPT-5.4"},
    {ID: "gpt-5.4-mini", Object: "model", Type: "model", DisplayName: "GPT-5.4 mini"},
    {ID: "gpt-5.5", Object: "model", Type: "model", DisplayName: "GPT-5.5"},
    {ID: "gemini-2.5-pro", Object: "model", Type: "model", DisplayName: "Gemini 2.5 Pro"},
    {ID: "gemini-3-flash-preview", Object: "model", Type: "model", DisplayName: "Gemini 3 Flash (Preview)"},
    {ID: "gemini-3.1-pro-preview", Object: "model", Type: "model", DisplayName: "Gemini 3.1 Pro (Preview)"},
    {ID: "grok-code-fast-1", Object: "model", Type: "model", DisplayName: "Grok Code Fast 1"},
    {ID: "gpt-4.1", Object: "model", Type: "model", DisplayName: "GPT-4.1"},
    {ID: "gpt-4o", Object: "model", Type: "model", DisplayName: "GPT-4o"},
    {ID: "gpt-5-mini", Object: "model", Type: "model", DisplayName: "GPT-5 mini"},
}
```

**Step 4: Add Copilot branch to available models**

In `backend/internal/handler/admin/account_handler.go`, import `internal/pkg/copilot` and add before the Gemini/Anthropic fallback:

```go
if account.Platform == service.PlatformCopilot {
    mapping := account.GetModelMapping()
    if len(mapping) == 0 {
        response.Success(c, copilot.DefaultModels)
        return
    }

    var models []copilot.Model
    for requestedModel := range mapping {
        var found bool
        for _, dm := range copilot.DefaultModels {
            if dm.ID == requestedModel {
                models = append(models, dm)
                found = true
                break
            }
        }
        if !found {
            models = append(models, copilot.Model{
                ID: requestedModel,
                Object: "model",
                Type: "model",
                DisplayName: requestedModel,
            })
        }
    }
    response.Success(c, models)
    return
}
```

**Step 5: Run tests and commit**

Run:

```bash
go test ./backend/internal/handler/admin -run 'TestAccountHandlerGetAvailableModels_(OpenAI|Copilot)' -count=1
```

Expected: PASS.

Commit:

```bash
git add backend/internal/handler/admin/account_handler.go backend/internal/handler/admin/account_handler_available_models_test.go backend/internal/pkg/copilot/models.go
git commit -m "fix: return copilot account models"
```

---

### Task 2: Backend Copilot account-test mapping

**Files:**
- Modify: `backend/internal/service/account_test_service.go`
- Test: create or modify `backend/internal/service/account_test_service_copilot_test.go`

**Step 1: Write failing test**

Create a test that confirms a Copilot account resolves `model_mapping` before dispatching to the Claude-compatible test path. Use the existing account-test service testing patterns in `backend/internal/service/account_test_service_openai_compact_test.go` and other `account_test_service_*` files.

Minimum assertion: the SSE `test_start` event should report the mapped upstream model when request model is mapped.

**Step 2: Run failing test**

Run:

```bash
go test ./backend/internal/service -run 'TestAccountTestService_.*Copilot.*Mapping' -count=1
```

Expected: FAIL because Copilot does not apply `GetMappedModel`.

**Step 3: Implement minimal mapping**

In `TestAccountConnection`, before falling back to `testClaudeAccountConnection`, add Copilot mapping behavior or in `testClaudeAccountConnection` extend the condition:

```go
if account.Type == AccountTypeAPIKey || account.Platform == PlatformCopilot {
    testModelID = account.GetMappedModel(testModelID)
}
```

Prefer the smallest safe change in `testClaudeAccountConnection`, because Copilot currently uses that path.

**Step 4: Run tests and commit**

Run:

```bash
go test ./backend/internal/service -run 'TestAccountTestService_.*Copilot.*Mapping|TestAccountTestService_TestAccountConnection_OpenAICompact' -count=1
```

Expected: PASS.

Commit:

```bash
git add backend/internal/service/account_test_service.go backend/internal/service/account_test_service_copilot_test.go
git commit -m "fix: apply copilot model mapping during tests"
```

---

### Task 3: Frontend edit modal Copilot model controls

**Files:**
- Modify: `frontend/src/components/account/EditAccountModal.vue`
- Test: `frontend/src/components/account/__tests__/EditAccountModal.spec.ts`

**Step 1: Write failing frontend tests**

Add tests that mount `EditAccountModal` with a Copilot OAuth account:

```ts
it('shows model restriction controls for Copilot OAuth accounts', async () => {
  const account = buildAccount({
    platform: 'copilot',
    type: 'oauth',
    credentials: { model_mapping: { 'gpt-5.4': 'gpt-5.4' } }
  })
  const wrapper = mountEditAccountModal(account)
  expect(wrapper.text()).toContain('Model Whitelist')
  expect(wrapper.find('[data-testid="model-whitelist-value"]').text()).toContain('gpt-5.4')
})

it('saves Copilot OAuth model mapping from edit modal', async () => {
  const account = buildAccount({ platform: 'copilot', type: 'oauth', credentials: {} })
  const wrapper = mountEditAccountModal(account)
  await selectModelWhitelist(wrapper, ['gpt-5.4', 'claude-sonnet-4.5'])
  await wrapper.find('form').trigger('submit.prevent')
  expect(updateAccountMock).toHaveBeenCalledWith(account.id, expect.objectContaining({
    credentials: expect.objectContaining({
      model_mapping: {
        'gpt-5.4': 'gpt-5.4',
        'claude-sonnet-4.5': 'claude-sonnet-4.5'
      }
    })
  }))
})
```

Adapt helper names to existing test utilities in the file.

**Step 2: Run failing tests**

Run:

```bash
cd frontend && pnpm vitest run src/components/account/__tests__/EditAccountModal.spec.ts
```

Expected: FAIL because Copilot OAuth does not render/save model controls.

**Step 3: Implement UI and persistence**

In `EditAccountModal.vue`:

1. Add a Copilot OAuth model restriction section, mirroring OpenAI OAuth but with:

```vue
v-if="account.platform === 'copilot' && account.type === 'oauth'"
<ModelWhitelistSelector v-model="allowedModels" :platform="account?.platform || 'anthropic'" />
```

2. During account watcher initialization, ensure Copilot OAuth loads `credentials.model_mapping` into `modelRestrictionMode`, `allowedModels`, and `modelMappings`. This can be a shared helper to avoid duplicating OpenAI logic.

3. In `handleSubmit`, persist Copilot OAuth mapping:

```ts
if (props.account.platform === 'copilot' && props.account.type === 'oauth') {
  const currentCredentials = (updatePayload.credentials as Record<string, unknown>) ||
    ((props.account.credentials as Record<string, unknown>) || {})
  const newCredentials: Record<string, unknown> = { ...currentCredentials }
  const modelMapping = buildModelMappingObject(modelRestrictionMode.value, allowedModels.value, modelMappings.value)
  if (modelMapping) {
    newCredentials.model_mapping = modelMapping
  } else {
    delete newCredentials.model_mapping
  }
  updatePayload.credentials = newCredentials
}
```

**Step 4: Run tests and commit**

Run:

```bash
cd frontend && pnpm vitest run src/components/account/__tests__/EditAccountModal.spec.ts
```

Expected: PASS.

Commit:

```bash
git add frontend/src/components/account/EditAccountModal.vue frontend/src/components/account/__tests__/EditAccountModal.spec.ts
git commit -m "fix: edit copilot account model restrictions"
```

---

### Task 4: Frontend account test modal coverage

**Files:**
- Test: `frontend/src/components/account/__tests__/AccountTestModal.spec.ts`

**Step 1: Add/adjust test**

Add a Copilot account test case where `adminAPI.accounts.getAvailableModels` returns Copilot mapping models, then assert the select options include those IDs and no hardcoded Claude-only fallback is used.

**Step 2: Run test**

Run:

```bash
cd frontend && pnpm vitest run src/components/account/__tests__/AccountTestModal.spec.ts
```

Expected: PASS if component already trusts backend models. If it fails, fix only Copilot-specific assumptions in `AccountTestModal.vue`.

**Step 3: Commit if changed**

```bash
git add frontend/src/components/account/AccountTestModal.vue frontend/src/components/account/__tests__/AccountTestModal.spec.ts
git commit -m "test: cover copilot account test models"
```

---

### Task 5: Full verification

**Files:**
- No planned source edits unless tests reveal issues.

**Step 1: Run targeted backend tests**

```bash
go test ./backend/internal/handler/admin ./backend/internal/service -run 'Copilot|AvailableModels|AccountTestService' -count=1
```

Expected: PASS.

**Step 2: Run targeted frontend tests**

```bash
cd frontend && pnpm vitest run \
  src/components/account/__tests__/EditAccountModal.spec.ts \
  src/components/account/__tests__/AccountTestModal.spec.ts
```

Expected: PASS.

**Step 3: Inspect diff**

```bash
git status --short
git diff --stat HEAD~4..HEAD
```

Expected: only intended files changed.

**Step 4: Final summary**

Report:

- Commit hashes created.
- Tests run and results.
- Any remaining deployment step needed.

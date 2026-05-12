package copilot

// Model describes a GitHub Copilot model option exposed to the admin UI.
type Model struct {
	ID          string `json:"id"`
	Object      string `json:"object"`
	Type        string `json:"type"`
	DisplayName string `json:"display_name"`
}

// DefaultModels is the curated GitHub Copilot model list used when an account
// does not have an explicit model_mapping configured. Keep this list in sync
// with frontend/src/composables/useModelWhitelist.ts copilotModelMeta.
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

package copilot

import "sort"

// Model describes a GitHub Copilot model option exposed to the admin UI.
type Model struct {
	ID          string `json:"id"`
	Object      string `json:"object"`
	Type        string `json:"type"`
	DisplayName string `json:"display_name"`
	Tier        string `json:"tier,omitempty"`
	Multiplier  string `json:"multiplier,omitempty"`
}

var defaultModelMeta = map[string]Model{
	"claude-haiku-4.5":       {ID: "claude-haiku-4.5", Object: "model", Type: "model", DisplayName: "Claude Haiku 4.5", Tier: "premium", Multiplier: "0.33x"},
	"claude-opus-4.5":        {ID: "claude-opus-4.5", Object: "model", Type: "model", DisplayName: "Claude Opus 4.5", Tier: "premium", Multiplier: "3x"},
	"claude-opus-4.6":        {ID: "claude-opus-4.6", Object: "model", Type: "model", DisplayName: "Claude Opus 4.6", Tier: "premium", Multiplier: "3x"},
	"claude-opus-4.7":        {ID: "claude-opus-4.7", Object: "model", Type: "model", DisplayName: "Claude Opus 4.7", Tier: "premium", Multiplier: "15x"},
	"claude-sonnet-4.5":      {ID: "claude-sonnet-4.5", Object: "model", Type: "model", DisplayName: "Claude Sonnet 4.5", Tier: "premium", Multiplier: "1x"},
	"claude-sonnet-4.6":      {ID: "claude-sonnet-4.6", Object: "model", Type: "model", DisplayName: "Claude Sonnet 4.6", Tier: "premium", Multiplier: "1x"},
	"gpt-5.2":                {ID: "gpt-5.2", Object: "model", Type: "model", DisplayName: "GPT-5.2", Tier: "premium", Multiplier: "1x"},
	"gpt-5.2-codex":          {ID: "gpt-5.2-codex", Object: "model", Type: "model", DisplayName: "GPT-5.2-Codex", Tier: "premium", Multiplier: "1x"},
	"gpt-5.3-codex":          {ID: "gpt-5.3-codex", Object: "model", Type: "model", DisplayName: "GPT-5.3-Codex", Tier: "premium", Multiplier: "1x"},
	"gpt-5.4":                {ID: "gpt-5.4", Object: "model", Type: "model", DisplayName: "GPT-5.4", Tier: "premium", Multiplier: "1x"},
	"gpt-5.4-mini":           {ID: "gpt-5.4-mini", Object: "model", Type: "model", DisplayName: "GPT-5.4 mini", Tier: "premium", Multiplier: "0.33x"},
	"gpt-5.5":                {ID: "gpt-5.5", Object: "model", Type: "model", DisplayName: "GPT-5.5", Tier: "premium", Multiplier: "7.5x"},
	"gemini-2.5-pro":         {ID: "gemini-2.5-pro", Object: "model", Type: "model", DisplayName: "Gemini 2.5 Pro", Tier: "premium", Multiplier: "1x"},
	"gemini-3-flash-preview": {ID: "gemini-3-flash-preview", Object: "model", Type: "model", DisplayName: "Gemini 3 Flash (Preview)", Tier: "premium", Multiplier: "0.33x"},
	"gemini-3.1-pro-preview": {ID: "gemini-3.1-pro-preview", Object: "model", Type: "model", DisplayName: "Gemini 3.1 Pro (Preview)", Tier: "premium", Multiplier: "1x"},
	"grok-code-fast-1":       {ID: "grok-code-fast-1", Object: "model", Type: "model", DisplayName: "Grok Code Fast 1", Tier: "premium", Multiplier: "0.25x"},
	"gpt-4.1":                {ID: "gpt-4.1", Object: "model", Type: "model", DisplayName: "GPT-4.1", Tier: "standard", Multiplier: "included"},
	"gpt-4o":                 {ID: "gpt-4o", Object: "model", Type: "model", DisplayName: "GPT-4o", Tier: "standard", Multiplier: "included"},
	"gpt-5-mini":             {ID: "gpt-5-mini", Object: "model", Type: "model", DisplayName: "GPT-5 mini", Tier: "standard", Multiplier: "included"},
}

var defaultModelOrder = []string{
	"claude-haiku-4.5",
	"claude-opus-4.5",
	"claude-opus-4.6",
	"claude-opus-4.7",
	"claude-sonnet-4.5",
	"claude-sonnet-4.6",
	"gpt-5.2",
	"gpt-5.2-codex",
	"gpt-5.3-codex",
	"gpt-5.4",
	"gpt-5.4-mini",
	"gpt-5.5",
	"gemini-2.5-pro",
	"gemini-3-flash-preview",
	"gemini-3.1-pro-preview",
	"grok-code-fast-1",
	"gpt-4.1",
	"gpt-4o",
	"gpt-5-mini",
}

// DefaultModels returns the curated GitHub Copilot model list used when an account
// does not have an explicit model_mapping configured.
func DefaultModels() []Model {
	models := make([]Model, 0, len(defaultModelOrder))
	for _, id := range defaultModelOrder {
		models = append(models, defaultModelMeta[id])
	}
	return models
}

// ModelsFromMapping returns Copilot model metadata for the requested mapping keys.
func ModelsFromMapping(mapping map[string]string) []Model {
	ids := make([]string, 0, len(mapping))
	for id := range mapping {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	models := make([]Model, 0, len(ids))
	for _, id := range ids {
		if model, ok := defaultModelMeta[id]; ok {
			models = append(models, model)
			continue
		}
		models = append(models, Model{ID: id, Object: "model", Type: "model", DisplayName: id})
	}
	return models
}

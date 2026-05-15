package service

// InboundProtocol identifies the client-facing API protocol for capability-aware scheduling.
type InboundProtocol string

const (
	InboundProtocolAnthropicMessages       InboundProtocol = "anthropic_messages"
	InboundProtocolAnthropicCountTokens    InboundProtocol = "anthropic_count_tokens"
	InboundProtocolOpenAIChatCompletions   InboundProtocol = "openai_chat_completions"
	InboundProtocolOpenAIResponses         InboundProtocol = "openai_responses"
	InboundProtocolOpenAIImagesGenerations InboundProtocol = "openai_images_generations"
	InboundProtocolOpenAIImagesEdits       InboundProtocol = "openai_images_edits"
	InboundProtocolGeminiV1Beta            InboundProtocol = "gemini_v1beta"
	InboundProtocolCodexResponses          InboundProtocol = "codex_responses"
)

// AccountSupportsInboundProtocol reports whether account can satisfy protocol either natively
// or through an existing compatibility converter.
func AccountSupportsInboundProtocol(account *Account, protocol InboundProtocol) bool {
	if account == nil {
		return false
	}
	switch protocol {
	case InboundProtocolAnthropicMessages:
		switch account.Platform {
		case PlatformAnthropic, PlatformOpenAI, PlatformGemini, PlatformAntigravity:
			return true
		default:
			return false
		}
	case InboundProtocolAnthropicCountTokens:
		return account.Platform == PlatformAnthropic
	case InboundProtocolOpenAIChatCompletions:
		return account.Platform == PlatformOpenAI || account.Platform == PlatformCopilot || account.Platform == PlatformAnthropic
	case InboundProtocolOpenAIResponses:
		return account.Platform == PlatformOpenAI || account.Platform == PlatformAnthropic
	case InboundProtocolOpenAIImagesGenerations, InboundProtocolOpenAIImagesEdits:
		return account.Platform == PlatformOpenAI
	case InboundProtocolGeminiV1Beta:
		return account.Platform == PlatformGemini
	case InboundProtocolCodexResponses:
		return (account.Platform == PlatformOpenAI && account.Type == AccountTypeOAuth) || account.Platform == PlatformCopilot
	default:
		return false
	}
}

package service

import "testing"

func TestAccountSupportsInboundProtocol(t *testing.T) {
	tests := []struct {
		name     string
		account  Account
		protocol InboundProtocol
		want     bool
	}{
		{"anthropic_native_messages", Account{Platform: PlatformAnthropic}, InboundProtocolAnthropicMessages, true},
		{"openai_messages_compat", Account{Platform: PlatformOpenAI}, InboundProtocolAnthropicMessages, true},
		{"gemini_messages_compat", Account{Platform: PlatformGemini}, InboundProtocolAnthropicMessages, true},
		{"antigravity_messages", Account{Platform: PlatformAntigravity}, InboundProtocolAnthropicMessages, true},
		{"openai_native_chat", Account{Platform: PlatformOpenAI}, InboundProtocolOpenAIChatCompletions, true},
		{"copilot_native_chat", Account{Platform: PlatformCopilot}, InboundProtocolOpenAIChatCompletions, true},
		{"anthropic_chat_compat", Account{Platform: PlatformAnthropic}, InboundProtocolOpenAIChatCompletions, true},
		{"openai_native_responses", Account{Platform: PlatformOpenAI}, InboundProtocolOpenAIResponses, true},
		{"anthropic_responses_compat", Account{Platform: PlatformAnthropic}, InboundProtocolOpenAIResponses, true},
		{"openai_images", Account{Platform: PlatformOpenAI}, InboundProtocolOpenAIImagesGenerations, true},
		{"anthropic_no_images", Account{Platform: PlatformAnthropic}, InboundProtocolOpenAIImagesGenerations, false},
		{"gemini_native", Account{Platform: PlatformGemini}, InboundProtocolGeminiV1Beta, true},
		{"openai_no_gemini", Account{Platform: PlatformOpenAI}, InboundProtocolGeminiV1Beta, false},
		{"codex_openai_oauth", Account{Platform: PlatformOpenAI, Type: AccountTypeOAuth}, InboundProtocolCodexResponses, true},
		{"codex_openai_apikey_false", Account{Platform: PlatformOpenAI, Type: AccountTypeAPIKey}, InboundProtocolCodexResponses, false},
		{"codex_copilot", Account{Platform: PlatformCopilot, Type: AccountTypeOAuth}, InboundProtocolCodexResponses, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := AccountSupportsInboundProtocol(&tt.account, tt.protocol); got != tt.want {
				t.Fatalf("AccountSupportsInboundProtocol() = %v, want %v", got, tt.want)
			}
		})
	}
}

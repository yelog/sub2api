package service

import "testing"

func TestAccountOpenAIFastPassthroughEnabled(t *testing.T) {
	tests := []struct {
		name    string
		account *Account
		want    bool
	}{
		{
			name:    "nil account defaults disabled",
			account: nil,
			want:    false,
		},
		{
			name: "nil extra defaults disabled",
			account: &Account{
				Platform: PlatformOpenAI,
				Extra:    nil,
			},
			want: false,
		},
		{
			name: "missing extra key defaults disabled",
			account: &Account{
				Platform: PlatformOpenAI,
				Extra:    map[string]any{"other": true},
			},
			want: false,
		},
		{
			name: "openai true enables passthrough",
			account: &Account{
				Platform: PlatformOpenAI,
				Extra: map[string]any{
					AccountExtraOpenAIFastPassthroughEnabled: true,
				},
			},
			want: true,
		},
		{
			name: "openai false disables passthrough",
			account: &Account{
				Platform: PlatformOpenAI,
				Extra: map[string]any{
					AccountExtraOpenAIFastPassthroughEnabled: false,
				},
			},
			want: false,
		},
		{
			name: "openai string true does not enable passthrough",
			account: &Account{
				Platform: PlatformOpenAI,
				Extra: map[string]any{
					AccountExtraOpenAIFastPassthroughEnabled: "true",
				},
			},
			want: false,
		},
		{
			name: "non openai true still disables passthrough",
			account: &Account{
				Platform: PlatformAnthropic,
				Extra: map[string]any{
					AccountExtraOpenAIFastPassthroughEnabled: true,
				},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.account.OpenAIFastPassthroughEnabled(); got != tt.want {
				t.Fatalf("OpenAIFastPassthroughEnabled() = %v, want %v", got, tt.want)
			}
		})
	}
}

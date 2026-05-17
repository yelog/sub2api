//go:build unit

package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

type openAIFastPassthroughAccountRepo struct {
	accountRepoStub
	created *Account
	updated *Account
	stored  *Account
}

func (r *openAIFastPassthroughAccountRepo) Create(_ context.Context, account *Account) error {
	r.created = account
	account.ID = 1
	return nil
}

func (r *openAIFastPassthroughAccountRepo) GetByID(_ context.Context, id int64) (*Account, error) {
	return r.stored, nil
}

func (r *openAIFastPassthroughAccountRepo) Update(_ context.Context, account *Account) error {
	r.updated = account
	return nil
}

func TestAdminServiceCreateAccountStoresOpenAIFastPassthroughFlag(t *testing.T) {
	svc := &adminServiceImpl{accountRepo: &openAIFastPassthroughAccountRepo{}}

	account, err := svc.CreateAccount(context.Background(), &CreateAccountInput{
		Name:                         "openai",
		Platform:                     PlatformOpenAI,
		Type:                         AccountTypeAPIKey,
		Credentials:                  map[string]any{"api_key": "sk-test"},
		OpenAIFastPassthroughEnabled: boolPtrForOpenAIFastPassthroughTest(true),
		SkipDefaultGroupBind:         true,
		SkipMixedChannelCheck:        true,
	})

	require.NoError(t, err)
	require.True(t, account.OpenAIFastPassthroughEnabled())
	require.Equal(t, true, account.Extra[AccountExtraOpenAIFastPassthroughEnabled])
}

func TestAdminServiceCreateAccountDefaultsOpenAIFastPassthroughOff(t *testing.T) {
	svc := &adminServiceImpl{accountRepo: &openAIFastPassthroughAccountRepo{}}

	account, err := svc.CreateAccount(context.Background(), &CreateAccountInput{
		Name:                  "openai",
		Platform:              PlatformOpenAI,
		Type:                  AccountTypeAPIKey,
		Credentials:           map[string]any{"api_key": "sk-test"},
		SkipDefaultGroupBind:  true,
		SkipMixedChannelCheck: true,
	})

	require.NoError(t, err)
	require.False(t, account.OpenAIFastPassthroughEnabled())
	require.Equal(t, false, account.Extra[AccountExtraOpenAIFastPassthroughEnabled])
}

func TestAdminServiceUpdateAccountCanSetOpenAIFastPassthroughFalse(t *testing.T) {
	repo := &openAIFastPassthroughAccountRepo{
		stored: &Account{
			ID:       1,
			Platform: PlatformOpenAI,
			Type:     AccountTypeAPIKey,
			Extra: map[string]any{
				AccountExtraOpenAIFastPassthroughEnabled: true,
				"quota_used":                             12.3,
			},
		},
	}
	svc := &adminServiceImpl{accountRepo: repo}

	account, err := svc.UpdateAccount(context.Background(), 1, &UpdateAccountInput{
		OpenAIFastPassthroughEnabled: boolPtrForOpenAIFastPassthroughTest(false),
	})

	require.NoError(t, err)
	require.False(t, account.OpenAIFastPassthroughEnabled())
	require.Equal(t, false, account.Extra[AccountExtraOpenAIFastPassthroughEnabled])
	require.Equal(t, 12.3, account.Extra["quota_used"])
	require.Same(t, account, repo.updated)
}

func TestAdminServiceUpdateAccountMissingOpenAIFastPassthroughLeavesExistingValue(t *testing.T) {
	repo := &openAIFastPassthroughAccountRepo{
		stored: &Account{
			ID:       1,
			Platform: PlatformOpenAI,
			Type:     AccountTypeAPIKey,
			Extra: map[string]any{
				AccountExtraOpenAIFastPassthroughEnabled: true,
			},
		},
	}
	svc := &adminServiceImpl{accountRepo: repo}

	account, err := svc.UpdateAccount(context.Background(), 1, &UpdateAccountInput{})

	require.NoError(t, err)
	require.True(t, account.OpenAIFastPassthroughEnabled())
	require.Equal(t, true, account.Extra[AccountExtraOpenAIFastPassthroughEnabled])
}

func TestAdminServiceCreateAccountIgnoresOpenAIFastPassthroughForNonOpenAI(t *testing.T) {
	svc := &adminServiceImpl{accountRepo: &openAIFastPassthroughAccountRepo{}}

	account, err := svc.CreateAccount(context.Background(), &CreateAccountInput{
		Name:                         "anthropic",
		Platform:                     PlatformAnthropic,
		Type:                         AccountTypeAPIKey,
		Credentials:                  map[string]any{"api_key": "sk-ant"},
		OpenAIFastPassthroughEnabled: boolPtrForOpenAIFastPassthroughTest(true),
		SkipDefaultGroupBind:         true,
		SkipMixedChannelCheck:        true,
	})

	require.NoError(t, err)
	require.False(t, account.OpenAIFastPassthroughEnabled())
	require.NotContains(t, account.Extra, AccountExtraOpenAIFastPassthroughEnabled)
}

func boolPtrForOpenAIFastPassthroughTest(v bool) *bool { return &v }

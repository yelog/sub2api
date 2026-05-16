//go:build unit

package service

import (
	"context"
	"errors"
	"testing"
)

type accountRepoStubForBillingArchitecture struct {
	AccountRepository
	account *Account
	err     error
}

func (s *accountRepoStubForBillingArchitecture) GetByID(_ context.Context, id int64) (*Account, error) {
	if s.err != nil {
		return nil, s.err
	}
	if s.account == nil || s.account.ID != id {
		return nil, ErrAccountNotFound
	}
	return s.account, nil
}

type userGroupRateRepoStubForBillingArchitecture struct {
	UserGroupRateRepository
	rate *float64
	err  error
}

func (s *userGroupRateRepoStubForBillingArchitecture) GetByUserAndGroup(_ context.Context, _, _ int64) (*float64, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.rate, nil
}

func TestGetAccountBillingArchitecture_AccountRateMultiplierNilDefaultsToOne(t *testing.T) {
	t.Parallel()

	svc := &adminServiceImpl{
		accountRepo: &accountRepoStubForBillingArchitecture{account: &Account{
			ID:       10,
			Platform: PlatformAnthropic,
			Type:     AccountTypeOAuth,
		}},
	}

	arch, err := svc.GetAccountBillingArchitecture(context.Background(), 10, BillingArchitectureOptions{})
	if err != nil {
		t.Fatalf("GetAccountBillingArchitecture returned error: %v", err)
	}
	if arch.AccountRateMultiplier != nil {
		t.Fatalf("AccountRateMultiplier = %v, want nil", *arch.AccountRateMultiplier)
	}
	if arch.EffectiveAccountRateMultiplier != 1.0 {
		t.Fatalf("EffectiveAccountRateMultiplier = %v, want 1", arch.EffectiveAccountRateMultiplier)
	}
}

func TestGetAccountBillingArchitecture_AccountRateMultiplierZeroAllowed(t *testing.T) {
	t.Parallel()

	zero := 0.0
	svc := &adminServiceImpl{
		accountRepo: &accountRepoStubForBillingArchitecture{account: &Account{
			ID:             11,
			Platform:       PlatformOpenAI,
			Type:           AccountTypeAPIKey,
			RateMultiplier: &zero,
		}},
	}

	arch, err := svc.GetAccountBillingArchitecture(context.Background(), 11, BillingArchitectureOptions{})
	if err != nil {
		t.Fatalf("GetAccountBillingArchitecture returned error: %v", err)
	}
	if arch.AccountRateMultiplier == nil || *arch.AccountRateMultiplier != 0 {
		t.Fatalf("AccountRateMultiplier = %v, want 0", arch.AccountRateMultiplier)
	}
	if arch.EffectiveAccountRateMultiplier != 0 {
		t.Fatalf("EffectiveAccountRateMultiplier = %v, want 0", arch.EffectiveAccountRateMultiplier)
	}
}

func TestGetAccountBillingArchitecture_UserGroupOverrideWins(t *testing.T) {
	t.Parallel()

	accountRate := 1.1
	groupRate := 1.5
	userRate := 0.8
	svc := &adminServiceImpl{
		accountRepo: &accountRepoStubForBillingArchitecture{account: &Account{
			ID:             12,
			Platform:       PlatformAnthropic,
			Type:           AccountTypeAPIKey,
			RateMultiplier: &accountRate,
			Groups: []*Group{
				{ID: 101, Name: "anthropic-pro", Platform: PlatformAnthropic, RateMultiplier: groupRate, SubscriptionType: SubscriptionTypeSubscription},
			},
		}},
		userGroupRateRepo: &userGroupRateRepoStubForBillingArchitecture{rate: &userRate},
	}

	arch, err := svc.GetAccountBillingArchitecture(context.Background(), 12, BillingArchitectureOptions{UserID: 7, GroupID: 101})
	if err != nil {
		t.Fatalf("GetAccountBillingArchitecture returned error: %v", err)
	}
	if arch.EffectiveUserRateMultiplier == nil || *arch.EffectiveUserRateMultiplier != userRate {
		t.Fatalf("EffectiveUserRateMultiplier = %v, want %v", arch.EffectiveUserRateMultiplier, userRate)
	}
	if arch.UserRateSource != UserRateSourceUserGroupOverride {
		t.Fatalf("UserRateSource = %q, want %q", arch.UserRateSource, UserRateSourceUserGroupOverride)
	}
}

func TestGetAccountBillingArchitecture_GroupMultiplierFallback(t *testing.T) {
	t.Parallel()

	groupRate := 1.25
	svc := &adminServiceImpl{
		accountRepo: &accountRepoStubForBillingArchitecture{account: &Account{
			ID:       13,
			Platform: PlatformOpenAI,
			Type:     AccountTypeOAuth,
			Groups: []*Group{
				{ID: 201, Name: "openai-default", Platform: PlatformOpenAI, RateMultiplier: groupRate, SubscriptionType: SubscriptionTypeStandard},
			},
		}},
		userGroupRateRepo: &userGroupRateRepoStubForBillingArchitecture{},
	}

	arch, err := svc.GetAccountBillingArchitecture(context.Background(), 13, BillingArchitectureOptions{UserID: 8, GroupID: 201})
	if err != nil {
		t.Fatalf("GetAccountBillingArchitecture returned error: %v", err)
	}
	if arch.EffectiveUserRateMultiplier == nil || *arch.EffectiveUserRateMultiplier != groupRate {
		t.Fatalf("EffectiveUserRateMultiplier = %v, want %v", arch.EffectiveUserRateMultiplier, groupRate)
	}
	if arch.UserRateSource != UserRateSourceGroupDefault {
		t.Fatalf("UserRateSource = %q, want %q", arch.UserRateSource, UserRateSourceGroupDefault)
	}
}

func TestGetAccountBillingArchitecture_CostSemanticsStable(t *testing.T) {
	t.Parallel()

	svc := &adminServiceImpl{
		accountRepo: &accountRepoStubForBillingArchitecture{account: &Account{ID: 14, Platform: PlatformAnthropic, Type: AccountTypeOAuth}},
	}

	arch, err := svc.GetAccountBillingArchitecture(context.Background(), 14, BillingArchitectureOptions{})
	if err != nil {
		t.Fatalf("GetAccountBillingArchitecture returned error: %v", err)
	}

	want := map[string]string{
		"total_cost":              "standard_cost_before_user_multiplier",
		"actual_cost":             "cost_after_group_or_user_group_multiplier",
		"balance_cost":            "actual_cost",
		"subscription_cost":       "actual_cost",
		"api_key_quota_cost":      "actual_cost",
		"api_key_rate_limit_cost": "actual_cost",
		"account_quota_cost":      "total_cost * account_rate_multiplier",
	}
	for key, value := range want {
		if arch.CostSemantics[key] != value {
			t.Fatalf("CostSemantics[%q] = %q, want %q", key, arch.CostSemantics[key], value)
		}
	}
}

func TestGetAccountBillingArchitecture_ReturnsRepoErrors(t *testing.T) {
	t.Parallel()

	boom := errors.New("boom")
	svc := &adminServiceImpl{accountRepo: &accountRepoStubForBillingArchitecture{err: boom}}
	_, err := svc.GetAccountBillingArchitecture(context.Background(), 99, BillingArchitectureOptions{})
	if !errors.Is(err, boom) {
		t.Fatalf("error = %v, want %v", err, boom)
	}
}

func TestBuildUsageBillingCommand_BalanceUsesActualCostAndAccountQuotaUsesAccountMultiplier(t *testing.T) {
	t.Parallel()

	cmd := buildUsageBillingCommand("req-balance", nil, &postUsageBillingParams{
		Cost:   &CostBreakdown{TotalCost: 10, ActualCost: 15},
		User:   &User{ID: 1},
		APIKey: &APIKey{ID: 2, Quota: 100},
		Account: &Account{ID: 3, Type: AccountTypeAPIKey, Extra: map[string]any{
			"quota_limit": 100.0,
		}},
		AccountRateMultiplier: 0.5,
		APIKeyService:         noopAPIKeyQuotaUpdaterForBillingArchitecture{},
	})
	if cmd == nil {
		t.Fatal("buildUsageBillingCommand returned nil")
	}
	if cmd.BalanceCost != 15 {
		t.Fatalf("BalanceCost = %v, want 15", cmd.BalanceCost)
	}
	if cmd.SubscriptionCost != 0 {
		t.Fatalf("SubscriptionCost = %v, want 0", cmd.SubscriptionCost)
	}
	if cmd.APIKeyQuotaCost != 15 {
		t.Fatalf("APIKeyQuotaCost = %v, want 15", cmd.APIKeyQuotaCost)
	}
	if cmd.AccountQuotaCost != 5 {
		t.Fatalf("AccountQuotaCost = %v, want 5", cmd.AccountQuotaCost)
	}
}

func TestBuildUsageBillingCommand_SubscriptionUsesActualCost(t *testing.T) {
	t.Parallel()

	cmd := buildUsageBillingCommand("req-sub", nil, &postUsageBillingParams{
		Cost:                  &CostBreakdown{TotalCost: 10, ActualCost: 20},
		User:                  &User{ID: 1},
		APIKey:                &APIKey{ID: 2},
		Account:               &Account{ID: 3, Type: AccountTypeOAuth},
		Subscription:          &UserSubscription{ID: 4},
		IsSubscriptionBill:    true,
		AccountRateMultiplier: 2,
	})
	if cmd == nil {
		t.Fatal("buildUsageBillingCommand returned nil")
	}
	if cmd.BalanceCost != 0 {
		t.Fatalf("BalanceCost = %v, want 0", cmd.BalanceCost)
	}
	if cmd.SubscriptionCost != 20 {
		t.Fatalf("SubscriptionCost = %v, want 20", cmd.SubscriptionCost)
	}
	if cmd.AccountQuotaCost != 0 {
		t.Fatalf("AccountQuotaCost = %v, want 0", cmd.AccountQuotaCost)
	}
}

type noopAPIKeyQuotaUpdaterForBillingArchitecture struct{}

func (noopAPIKeyQuotaUpdaterForBillingArchitecture) UpdateQuotaUsed(context.Context, int64, float64) error {
	return nil
}

func (noopAPIKeyQuotaUpdaterForBillingArchitecture) UpdateRateLimitUsage(context.Context, int64, float64) error {
	return nil
}

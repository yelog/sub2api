package service

import "context"

const (
	UserRateSourceSystemDefault     = "system_default"
	UserRateSourceGroupDefault      = "group_default"
	UserRateSourceUserGroupOverride = "user_group_override"
)

type BillingArchitectureOptions struct {
	UserID  int64
	GroupID int64
}

type AccountBillingArchitecture struct {
	AccountID                      int64                             `json:"account_id"`
	Platform                       string                            `json:"platform"`
	Type                           string                            `json:"type"`
	AccountRateMultiplier          *float64                          `json:"account_rate_multiplier,omitempty"`
	EffectiveAccountRateMultiplier float64                           `json:"effective_account_rate_multiplier"`
	Groups                         []AccountBillingArchitectureGroup `json:"groups"`
	EffectiveUserRateMultiplier    *float64                          `json:"effective_user_rate_multiplier,omitempty"`
	UserRateSource                 string                            `json:"user_rate_source,omitempty"`
	CostSemantics                  map[string]string                 `json:"cost_semantics"`
}

type AccountBillingArchitectureGroup struct {
	ID               int64   `json:"id"`
	Name             string  `json:"name"`
	Platform         string  `json:"platform"`
	RateMultiplier   float64 `json:"rate_multiplier"`
	SubscriptionType string  `json:"subscription_type"`
}

func (s *adminServiceImpl) GetAccountBillingArchitecture(ctx context.Context, accountID int64, opts BillingArchitectureOptions) (*AccountBillingArchitecture, error) {
	account, err := s.accountRepo.GetByID(ctx, accountID)
	if err != nil {
		return nil, err
	}
	if account == nil {
		return nil, ErrAccountNotFound
	}

	arch := NewAccountBillingArchitecture(account)

	if opts.UserID > 0 && opts.GroupID > 0 {
		rate, source, err := s.resolveBillingArchitectureUserRate(ctx, account, opts.UserID, opts.GroupID)
		if err != nil {
			return nil, err
		}
		arch.EffectiveUserRateMultiplier = &rate
		arch.UserRateSource = source
	}

	return arch, nil
}

func NewAccountBillingArchitecture(account *Account) *AccountBillingArchitecture {
	if account == nil {
		return nil
	}
	arch := &AccountBillingArchitecture{
		AccountID:                      account.ID,
		Platform:                       account.Platform,
		Type:                           account.Type,
		AccountRateMultiplier:          account.RateMultiplier,
		EffectiveAccountRateMultiplier: account.BillingRateMultiplier(),
		Groups:                         buildAccountBillingArchitectureGroups(account),
		CostSemantics:                  defaultBillingArchitectureCostSemantics(),
	}
	return arch
}

func buildAccountBillingArchitectureGroups(account *Account) []AccountBillingArchitectureGroup {
	if account == nil || len(account.Groups) == 0 {
		return nil
	}
	groups := make([]AccountBillingArchitectureGroup, 0, len(account.Groups))
	for _, group := range account.Groups {
		if group == nil {
			continue
		}
		groups = append(groups, AccountBillingArchitectureGroup{
			ID:               group.ID,
			Name:             group.Name,
			Platform:         group.Platform,
			RateMultiplier:   group.RateMultiplier,
			SubscriptionType: group.SubscriptionType,
		})
	}
	return groups
}

func (s *adminServiceImpl) resolveBillingArchitectureUserRate(ctx context.Context, account *Account, userID, groupID int64) (float64, string, error) {
	if s.userGroupRateRepo != nil {
		rate, err := s.userGroupRateRepo.GetByUserAndGroup(ctx, userID, groupID)
		if err != nil {
			return 0, "", err
		}
		if rate != nil {
			return *rate, UserRateSourceUserGroupOverride, nil
		}
	}

	if account != nil {
		for _, group := range account.Groups {
			if group != nil && group.ID == groupID {
				return group.RateMultiplier, UserRateSourceGroupDefault, nil
			}
		}
	}

	return 1.0, UserRateSourceSystemDefault, nil
}

func defaultBillingArchitectureCostSemantics() map[string]string {
	return map[string]string{
		"total_cost":              "standard_cost_before_user_multiplier",
		"actual_cost":             "cost_after_group_or_user_group_multiplier",
		"balance_cost":            "actual_cost",
		"subscription_cost":       "actual_cost",
		"api_key_quota_cost":      "actual_cost",
		"api_key_rate_limit_cost": "actual_cost",
		"account_quota_cost":      "total_cost * account_rate_multiplier",
	}
}

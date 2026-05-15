package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
)

func TestSelectAccountForProtocolWithLoadAwarenessFiltersByCapability(t *testing.T) {
	groupID := int64(10)
	svc := newProtocolSelectionGatewayServiceForTest([]Account{
		protocolSelectionAccount(1, PlatformAnthropic, groupID),
		protocolSelectionAccount(2, PlatformOpenAI, groupID),
	})

	got, err := svc.SelectAccountForProtocolWithLoadAwareness(context.Background(), &groupID, InboundProtocolOpenAIImagesGenerations, "", "gpt-image-1", nil, "", 0)
	if err != nil {
		t.Fatalf("SelectAccountForProtocolWithLoadAwareness() error = %v", err)
	}
	if got == nil || got.Account == nil || got.Account.ID != 2 {
		t.Fatalf("selected account = %#v, want account 2", got)
	}
}

func TestSelectAccountForProtocolWithLoadAwarenessCanSelectOpenAIForMessages(t *testing.T) {
	groupID := int64(10)
	svc := newProtocolSelectionGatewayServiceForTest([]Account{
		protocolSelectionAccount(2, PlatformOpenAI, groupID),
	})

	got, err := svc.SelectAccountForProtocolWithLoadAwareness(context.Background(), &groupID, InboundProtocolAnthropicMessages, "", "gpt-5.5", nil, "", 0)
	if err != nil {
		t.Fatalf("SelectAccountForProtocolWithLoadAwareness() error = %v", err)
	}
	if got == nil || got.Account == nil || got.Account.ID != 2 {
		t.Fatalf("selected account = %#v, want account 2", got)
	}
}

func TestSelectAccountForProtocolWithLoadAwarenessNoCapableAccount(t *testing.T) {
	groupID := int64(10)
	svc := newProtocolSelectionGatewayServiceForTest([]Account{
		protocolSelectionAccount(2, PlatformOpenAI, groupID),
	})

	_, err := svc.SelectAccountForProtocolWithLoadAwareness(context.Background(), &groupID, InboundProtocolGeminiV1Beta, "", "gemini-2.5-pro", nil, "", 0)
	if !errors.Is(err, ErrNoAvailableAccounts) {
		t.Fatalf("error = %v, want ErrNoAvailableAccounts", err)
	}
}

func TestSelectAccountForProtocolWithLoadAwarenessHonorsForcePlatform(t *testing.T) {
	groupID := int64(10)
	svc := newProtocolSelectionGatewayServiceForTest([]Account{
		protocolSelectionAccount(1, PlatformAnthropic, groupID),
		protocolSelectionAccount(2, PlatformOpenAI, groupID),
	})

	ctx := context.WithValue(context.Background(), ctxkey.ForcePlatform, PlatformAnthropic)
	_, err := svc.SelectAccountForProtocolWithLoadAwareness(ctx, &groupID, InboundProtocolOpenAIImagesGenerations, "", "gpt-image-1", nil, "", 0)
	if !errors.Is(err, ErrNoAvailableAccounts) {
		t.Fatalf("error = %v, want ErrNoAvailableAccounts", err)
	}
}

func newProtocolSelectionGatewayServiceForTest(accounts []Account) *GatewayService {
	return &GatewayService{
		accountRepo: &protocolSelectionAccountRepo{accounts: accounts},
		groupRepo: &protocolSelectionGroupRepo{groups: map[int64]*Group{
			10: {ID: 10, Name: "mixed", Platform: "legacy", Status: StatusActive},
		}},
		cfg: &config.Config{RunMode: config.RunModeStandard},
	}
}

func protocolSelectionAccount(id int64, platform string, groupID int64) Account {
	return Account{
		ID:            id,
		Name:          platform,
		Platform:      platform,
		Type:          AccountTypeOAuth,
		Status:        StatusActive,
		Schedulable:   true,
		Concurrency:   1,
		AccountGroups: []AccountGroup{{GroupID: groupID}},
		GroupIDs:      []int64{groupID},
	}
}

type protocolSelectionAccountRepo struct {
	accounts []Account
}

func (r *protocolSelectionAccountRepo) GetByID(ctx context.Context, id int64) (*Account, error) {
	for i := range r.accounts {
		if r.accounts[i].ID == id {
			return &r.accounts[i], nil
		}
	}
	return nil, errors.New("account not found")
}

func (r *protocolSelectionAccountRepo) GetByIDs(ctx context.Context, ids []int64) ([]*Account, error) {
	var result []*Account
	for _, id := range ids {
		if acc, err := r.GetByID(ctx, id); err == nil {
			result = append(result, acc)
		}
	}
	return result, nil
}

func (r *protocolSelectionAccountRepo) ExistsByID(ctx context.Context, id int64) (bool, error) {
	_, err := r.GetByID(ctx, id)
	if err != nil {
		return false, nil
	}
	return true, nil
}

func (r *protocolSelectionAccountRepo) ListSchedulableByGroupID(ctx context.Context, groupID int64) ([]Account, error) {
	var result []Account
	for _, acc := range r.accounts {
		if !acc.IsSchedulable() {
			continue
		}
		for _, ag := range acc.AccountGroups {
			if ag.GroupID == groupID {
				result = append(result, acc)
				break
			}
		}
	}
	return result, nil
}

func (r *protocolSelectionAccountRepo) ListSchedulableByGroupIDAndPlatform(ctx context.Context, groupID int64, platform string) ([]Account, error) {
	var result []Account
	accounts, err := r.ListSchedulableByGroupID(ctx, groupID)
	if err != nil {
		return nil, err
	}
	for _, acc := range accounts {
		if acc.Platform == platform {
			result = append(result, acc)
		}
	}
	return result, nil
}

func (r *protocolSelectionAccountRepo) ListSchedulableByGroupIDAndPlatforms(ctx context.Context, groupID int64, platforms []string) ([]Account, error) {
	allowed := make(map[string]struct{}, len(platforms))
	for _, platform := range platforms {
		allowed[platform] = struct{}{}
	}
	var result []Account
	accounts, err := r.ListSchedulableByGroupID(ctx, groupID)
	if err != nil {
		return nil, err
	}
	for _, acc := range accounts {
		if _, ok := allowed[acc.Platform]; ok {
			result = append(result, acc)
		}
	}
	return result, nil
}

type protocolSelectionGroupRepo struct {
	groups map[int64]*Group
}

func (r *protocolSelectionGroupRepo) GetByIDLite(ctx context.Context, id int64) (*Group, error) {
	if group, ok := r.groups[id]; ok {
		return group, nil
	}
	return nil, ErrGroupNotFound
}

func (r *protocolSelectionGroupRepo) GetByID(ctx context.Context, id int64) (*Group, error) {
	return r.GetByIDLite(ctx, id)
}

func (r *protocolSelectionGroupRepo) Create(ctx context.Context, group *Group) error { return nil }
func (r *protocolSelectionGroupRepo) Update(ctx context.Context, group *Group) error { return nil }
func (r *protocolSelectionGroupRepo) Delete(ctx context.Context, id int64) error     { return nil }
func (r *protocolSelectionGroupRepo) DeleteCascade(ctx context.Context, id int64) ([]int64, error) {
	return nil, nil
}
func (r *protocolSelectionGroupRepo) List(ctx context.Context, params pagination.PaginationParams) ([]Group, *pagination.PaginationResult, error) {
	return nil, nil, nil
}
func (r *protocolSelectionGroupRepo) ListWithFilters(ctx context.Context, params pagination.PaginationParams, platform, status, search string, isExclusive *bool) ([]Group, *pagination.PaginationResult, error) {
	return nil, nil, nil
}
func (r *protocolSelectionGroupRepo) ListActive(ctx context.Context) ([]Group, error) {
	return nil, nil
}
func (r *protocolSelectionGroupRepo) ListActiveByPlatform(ctx context.Context, platform string) ([]Group, error) {
	return nil, nil
}
func (r *protocolSelectionGroupRepo) ExistsByName(ctx context.Context, name string) (bool, error) {
	return false, nil
}
func (r *protocolSelectionGroupRepo) GetAccountCount(ctx context.Context, groupID int64) (int64, int64, error) {
	return 0, 0, nil
}
func (r *protocolSelectionGroupRepo) DeleteAccountGroupsByGroupID(ctx context.Context, groupID int64) (int64, error) {
	return 0, nil
}
func (r *protocolSelectionGroupRepo) GetAccountIDsByGroupIDs(ctx context.Context, groupIDs []int64) ([]int64, error) {
	return nil, nil
}
func (r *protocolSelectionGroupRepo) BindAccountsToGroup(ctx context.Context, groupID int64, accountIDs []int64) error {
	return nil
}
func (r *protocolSelectionGroupRepo) UpdateSortOrders(ctx context.Context, updates []GroupSortOrderUpdate) error {
	return nil
}

func (r *protocolSelectionAccountRepo) Create(ctx context.Context, account *Account) error {
	return nil
}
func (r *protocolSelectionAccountRepo) GetByCRSAccountID(ctx context.Context, crsAccountID string) (*Account, error) {
	return nil, nil
}
func (r *protocolSelectionAccountRepo) FindByExtraField(ctx context.Context, key string, value any) ([]Account, error) {
	return nil, nil
}
func (r *protocolSelectionAccountRepo) ListCRSAccountIDs(ctx context.Context) (map[string]int64, error) {
	return nil, nil
}
func (r *protocolSelectionAccountRepo) Update(ctx context.Context, account *Account) error {
	return nil
}
func (r *protocolSelectionAccountRepo) Delete(ctx context.Context, id int64) error { return nil }
func (r *protocolSelectionAccountRepo) List(ctx context.Context, params pagination.PaginationParams) ([]Account, *pagination.PaginationResult, error) {
	return nil, nil, nil
}
func (r *protocolSelectionAccountRepo) ListWithFilters(ctx context.Context, params pagination.PaginationParams, platform, accountType, status, search string, groupID int64, privacyMode string) ([]Account, *pagination.PaginationResult, error) {
	return nil, nil, nil
}
func (r *protocolSelectionAccountRepo) ListByGroup(ctx context.Context, groupID int64) ([]Account, error) {
	return nil, nil
}
func (r *protocolSelectionAccountRepo) ListActive(ctx context.Context) ([]Account, error) {
	return nil, nil
}
func (r *protocolSelectionAccountRepo) ListByPlatform(ctx context.Context, platform string) ([]Account, error) {
	return nil, nil
}
func (r *protocolSelectionAccountRepo) UpdateLastUsed(ctx context.Context, id int64) error {
	return nil
}
func (r *protocolSelectionAccountRepo) BatchUpdateLastUsed(ctx context.Context, updates map[int64]time.Time) error {
	return nil
}
func (r *protocolSelectionAccountRepo) SetError(ctx context.Context, id int64, errorMsg string) error {
	return nil
}
func (r *protocolSelectionAccountRepo) ClearError(ctx context.Context, id int64) error { return nil }
func (r *protocolSelectionAccountRepo) SetSchedulable(ctx context.Context, id int64, schedulable bool) error {
	return nil
}
func (r *protocolSelectionAccountRepo) AutoPauseExpiredAccounts(ctx context.Context, now time.Time) (int64, error) {
	return 0, nil
}
func (r *protocolSelectionAccountRepo) BindGroups(ctx context.Context, accountID int64, groupIDs []int64) error {
	return nil
}
func (r *protocolSelectionAccountRepo) ListSchedulable(ctx context.Context) ([]Account, error) {
	return r.accounts, nil
}
func (r *protocolSelectionAccountRepo) ListSchedulableByPlatform(ctx context.Context, platform string) ([]Account, error) {
	var result []Account
	for _, acc := range r.accounts {
		if acc.Platform == platform && acc.IsSchedulable() {
			result = append(result, acc)
		}
	}
	return result, nil
}
func (r *protocolSelectionAccountRepo) ListSchedulableByPlatforms(ctx context.Context, platforms []string) ([]Account, error) {
	allowed := make(map[string]struct{}, len(platforms))
	for _, platform := range platforms {
		allowed[platform] = struct{}{}
	}
	var result []Account
	for _, acc := range r.accounts {
		if _, ok := allowed[acc.Platform]; ok && acc.IsSchedulable() {
			result = append(result, acc)
		}
	}
	return result, nil
}
func (r *protocolSelectionAccountRepo) ListSchedulableUngroupedByPlatform(ctx context.Context, platform string) ([]Account, error) {
	return r.ListSchedulableByPlatform(ctx, platform)
}
func (r *protocolSelectionAccountRepo) ListSchedulableUngroupedByPlatforms(ctx context.Context, platforms []string) ([]Account, error) {
	return r.ListSchedulableByPlatforms(ctx, platforms)
}
func (r *protocolSelectionAccountRepo) SetRateLimited(ctx context.Context, id int64, resetAt time.Time) error {
	return nil
}
func (r *protocolSelectionAccountRepo) SetModelRateLimit(ctx context.Context, id int64, scope string, resetAt time.Time) error {
	return nil
}
func (r *protocolSelectionAccountRepo) SetOverloaded(ctx context.Context, id int64, until time.Time) error {
	return nil
}
func (r *protocolSelectionAccountRepo) SetTempUnschedulable(ctx context.Context, id int64, until time.Time, reason string) error {
	return nil
}
func (r *protocolSelectionAccountRepo) ClearTempUnschedulable(ctx context.Context, id int64) error {
	return nil
}
func (r *protocolSelectionAccountRepo) ClearRateLimit(ctx context.Context, id int64) error {
	return nil
}
func (r *protocolSelectionAccountRepo) ClearAntigravityQuotaScopes(ctx context.Context, id int64) error {
	return nil
}
func (r *protocolSelectionAccountRepo) ClearModelRateLimits(ctx context.Context, id int64) error {
	return nil
}
func (r *protocolSelectionAccountRepo) UpdateSessionWindow(ctx context.Context, id int64, start, end *time.Time, status string) error {
	return nil
}
func (r *protocolSelectionAccountRepo) UpdateExtra(ctx context.Context, id int64, updates map[string]any) error {
	return nil
}
func (r *protocolSelectionAccountRepo) BulkUpdate(ctx context.Context, ids []int64, updates AccountBulkUpdate) (int64, error) {
	return 0, nil
}
func (r *protocolSelectionAccountRepo) IncrementQuotaUsed(ctx context.Context, id int64, amount float64) error {
	return nil
}
func (r *protocolSelectionAccountRepo) ResetQuotaUsed(ctx context.Context, id int64) error {
	return nil
}

var _ AccountRepository = (*protocolSelectionAccountRepo)(nil)
var _ GroupRepository = (*protocolSelectionGroupRepo)(nil)

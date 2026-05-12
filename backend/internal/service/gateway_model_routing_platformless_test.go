package service

import (
	"context"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

func TestPlatformlessModelRoutingAppliesForNonAnthropicGroup(t *testing.T) {
	ctx := context.Background()
	groupID := int64(91020)
	svc := newPlatformlessModelRoutingGatewayServiceForTest(groupID, PlatformOpenAI, map[string][]int64{"gpt-5.5": {2}}, []Account{
		platformlessRoutingAccount(1, PlatformAnthropic, groupID),
		platformlessRoutingAccount(2, PlatformOpenAI, groupID),
	})

	selection, err := svc.SelectAccountForProtocolWithLoadAwareness(ctx, &groupID, InboundProtocolAnthropicMessages, "", "gpt-5.5", nil, "", 0)
	require.NoError(t, err)
	require.NotNil(t, selection)
	require.NotNil(t, selection.Account)
	require.Equal(t, int64(2), selection.Account.ID)
	if selection.ReleaseFunc != nil {
		selection.ReleaseFunc()
	}
}

func TestPlatformlessModelRoutingSkipsIncapableRoutedAccount(t *testing.T) {
	ctx := context.Background()
	groupID := int64(91021)
	svc := newPlatformlessModelRoutingGatewayServiceForTest(groupID, PlatformOpenAI, map[string][]int64{"gpt-image-1": {2}}, []Account{
		platformlessRoutingAccount(1, PlatformOpenAI, groupID),
		platformlessRoutingAccount(2, PlatformAnthropic, groupID),
	})

	selection, err := svc.SelectAccountForProtocolWithLoadAwareness(ctx, &groupID, InboundProtocolOpenAIImagesGenerations, "", "gpt-image-1", nil, "", 0)
	require.NoError(t, err)
	require.NotNil(t, selection)
	require.NotNil(t, selection.Account)
	require.Equal(t, int64(1), selection.Account.ID)
	if selection.ReleaseFunc != nil {
		selection.ReleaseFunc()
	}
}

func newPlatformlessModelRoutingGatewayServiceForTest(groupID int64, groupPlatform string, routing map[string][]int64, accounts []Account) *GatewayService {
	return &GatewayService{
		accountRepo: &protocolSelectionAccountRepo{accounts: accounts},
		groupRepo: &protocolSelectionGroupRepo{groups: map[int64]*Group{
			groupID: {
				ID:                  groupID,
				Name:                "platformless",
				Platform:            groupPlatform,
				Status:              StatusActive,
				ModelRoutingEnabled: true,
				ModelRouting:        routing,
			},
		}},
		concurrencyService: NewConcurrencyService(stubConcurrencyCache{}),
		cfg:                &config.Config{RunMode: config.RunModeStandard},
	}
}

func platformlessRoutingAccount(id int64, platform string, groupID int64) Account {
	return Account{
		ID:            id,
		Name:          platform,
		Platform:      platform,
		Type:          AccountTypeOAuth,
		Status:        StatusActive,
		Schedulable:   true,
		Concurrency:   1,
		Priority:      int(id),
		AccountGroups: []AccountGroup{{GroupID: groupID}},
		GroupIDs:      []int64{groupID},
	}
}

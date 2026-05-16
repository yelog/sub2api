import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import AccountBillingInfo from '../AccountBillingInfo.vue'
import type { AccountBillingArchitecture } from '@/types'

function makeArchitecture(overrides: Partial<AccountBillingArchitecture> = {}): AccountBillingArchitecture {
  return {
    account_id: 1,
    platform: 'anthropic',
    type: 'apikey',
    account_rate_multiplier: 1.5,
    effective_account_rate_multiplier: 1.5,
    groups: [
      {
        id: 10,
        name: 'anthropic-pro',
        platform: 'anthropic',
        rate_multiplier: 1.25,
        subscription_type: 'subscription'
      }
    ],
    cost_semantics: {
      total_cost: 'standard_cost_before_user_multiplier',
      actual_cost: 'cost_after_group_or_user_group_multiplier',
      balance_cost: 'actual_cost',
      subscription_cost: 'actual_cost',
      api_key_quota_cost: 'actual_cost',
      api_key_rate_limit_cost: 'actual_cost',
      account_quota_cost: 'total_cost * account_rate_multiplier'
    },
    ...overrides
  }
}

describe('AccountBillingInfo', () => {
  it('renders account multiplier and cost semantics for balance/subscription/quota', () => {
    const wrapper = mount(AccountBillingInfo, {
      props: { architecture: makeArchitecture() }
    })

    expect(wrapper.text()).toContain('计费架构')
    expect(wrapper.text()).toContain('账号倍率 ×1.5')
    expect(wrapper.text()).toContain('余额扣费')
    expect(wrapper.text()).toContain('ActualCost（分组/用户专属倍率后）')
    expect(wrapper.text()).toContain('订阅消耗')
    expect(wrapper.text()).toContain('API Key 配额')
    expect(wrapper.text()).toContain('账号配额')
    expect(wrapper.text()).toContain('TotalCost × 账号倍率')
  })

  it('renders group multiplier summary', () => {
    const wrapper = mount(AccountBillingInfo, {
      props: { architecture: makeArchitecture() }
    })

    expect(wrapper.text()).toContain('分组倍率')
    expect(wrapper.text()).toContain('anthropic-pro')
    expect(wrapper.text()).toContain('×1.25')
  })

  it('renders user-specific rate source when present', () => {
    const wrapper = mount(AccountBillingInfo, {
      props: {
        architecture: makeArchitecture({
          effective_user_rate_multiplier: 0.8,
          user_rate_source: 'user_group_override'
        })
      }
    })

    expect(wrapper.text()).toContain('用户有效倍率')
    expect(wrapper.text()).toContain('×0.8')
    expect(wrapper.text()).toContain('用户专属覆盖')
  })
})

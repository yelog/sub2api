import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'

import UsageFilters from '../UsageFilters.vue'

const messages: Record<string, string> = {
  'admin.usage.userFilter': 'User',
  'admin.usage.searchUserPlaceholder': 'Search users',
  'usage.apiKeyFilter': 'API key',
  'admin.usage.searchApiKeyPlaceholder': 'Search keys',
  'usage.model': 'Model',
  'admin.usage.account': 'Account',
  'admin.usage.searchAccountPlaceholder': 'Search accounts',
  'usage.type': 'Type',
  'admin.usage.billingType': 'Billing type',
  'admin.usage.billingMode': 'Billing mode',
  'admin.usage.group': 'Group',
  'usage.openaiFastFilter': 'OpenAI Fast',
  'usage.allOpenAIFast': 'All',
  'usage.openaiFast': 'Fast',
  'usage.openaiNonFast': 'Non-Fast',
  'common.refresh': 'Refresh',
  'common.reset': 'Reset',
  'admin.usage.cleanup.button': 'Cleanup',
  'usage.exportExcel': 'Export',
}

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => messages[key] ?? key,
    }),
  }
})

vi.mock('@/api/admin', () => ({
  adminAPI: {
    usage: {
      searchUsers: vi.fn(),
      searchApiKeys: vi.fn(),
    },
    accounts: {
      list: vi.fn(),
    },
    groups: {
      list: vi.fn().mockResolvedValue({ items: [] }),
    },
    dashboard: {
      getModelStats: vi.fn().mockResolvedValue({ models: [] }),
    },
  },
}))

const SelectStub = {
  props: ['modelValue', 'options'],
  emits: ['update:modelValue', 'change'],
  template: `
    <label class="select-stub">
      <span v-for="option in options" :key="String(option.value)">{{ option.label }}</span>
      <select
        :value="modelValue ?? ''"
        @change="onChange($event)"
      >
        <option v-for="option in options" :key="String(option.value)" :value="option.value ?? ''">
          {{ option.label }}
        </option>
      </select>
    </label>
  `,
  methods: {
    onChange(event: Event) {
      const raw = (event.target as HTMLSelectElement).value
      const value = raw === '' ? null : raw
      this.$emit('update:modelValue', value)
      this.$emit('change', value, null)
    },
  },
}

describe('UsageFilters OpenAI Fast filter', () => {
  it('renders all/fast/non-fast options and writes openai_fast to the model', async () => {
    const modelValue: Record<string, unknown> = { openai_fast: 'all' }
    const wrapper = mount(UsageFilters, {
      props: {
        modelValue,
        exporting: false,
        startDate: '2026-05-16',
        endDate: '2026-05-17',
      },
      global: {
        stubs: {
          Select: SelectStub,
        },
      },
    })

    expect(wrapper.text()).toContain('OpenAI Fast')
    expect(wrapper.text()).toContain('All')
    expect(wrapper.text()).toContain('Fast')
    expect(wrapper.text()).toContain('Non-Fast')

    const select = wrapper.find('[data-test="openai-fast-filter"] select')
    expect(select.exists()).toBe(true)
    await select.setValue('fast')

    expect(modelValue.openai_fast).toBe('fast')
    expect(wrapper.emitted('change')).toBeTruthy()
  })
})

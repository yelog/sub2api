import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import ModelWhitelistSelector from '../ModelWhitelistSelector.vue'

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showInfo: vi.fn()
  })
}))

vi.mock('@/api/admin/accounts', () => ({
  getAntigravityDefaultModelMapping: vi.fn()
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string, params?: Record<string, unknown>) => params ? `${key}:${JSON.stringify(params)}` : key
    })
  }
})

const mountSelector = (props: Record<string, unknown>) => mount(ModelWhitelistSelector, {
  props: {
    modelValue: [],
    ...props
  },
  global: {
    stubs: {
      ModelIcon: true,
      Icon: true
    }
  }
})

describe('ModelWhitelistSelector', () => {
  it('shows GitHub Copilot model labels and multipliers', async () => {
    const wrapper = mountSelector({ platform: 'copilot' })

    await wrapper.find('div.cursor-pointer').trigger('click')

    expect(wrapper.text()).toContain('Claude Opus 4.7')
    expect(wrapper.text()).toContain('15x')
    expect(wrapper.text()).toContain('GPT-4.1')
    expect(wrapper.text()).toContain('included')
  })

  it('does not show multiplier badges for non-Copilot platforms', async () => {
    const wrapper = mountSelector({ platform: 'openai' })

    await wrapper.find('div.cursor-pointer').trigger('click')

    expect(wrapper.text()).toContain('GPT-5.4')
    expect(wrapper.text()).not.toContain('included')
    expect(wrapper.text()).not.toContain('15x')
  })
})

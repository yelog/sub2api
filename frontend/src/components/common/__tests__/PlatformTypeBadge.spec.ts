import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import PlatformTypeBadge from '../PlatformTypeBadge.vue'
import { platformLabel } from '@/utils/platformColors'

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string) => key
  })
}))

describe('PlatformTypeBadge', () => {
  it('recognizes copilot platform label', () => {
    expect(platformLabel('copilot')).toBe('GitHub Copilot')
  })

  it('renders copilot badge without fallback style', () => {
    const wrapper = mount(PlatformTypeBadge, {
      props: {
        platform: 'copilot',
        type: 'oauth'
      },
      global: {
        stubs: {
          PlatformIcon: true,
          Icon: true
        }
      }
    })

    expect(wrapper.text()).toContain('GitHub Copilot')
    expect(wrapper.html()).toContain('bg-slate-100')
    expect(wrapper.html()).not.toContain('bg-blue-100')
  })
})

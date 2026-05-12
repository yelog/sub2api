import { describe, expect, it, vi } from 'vitest'

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({ showError: vi.fn() })
}))

vi.mock('vue-i18n', () => ({
  useI18n: () => ({ t: (key: string) => key })
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    copilot: {
      startDeviceFlow: vi.fn(),
      pollDeviceFlow: vi.fn()
    }
  }
}))

import { useCopilotOAuth } from '@/composables/useCopilotOAuth'
import { adminAPI } from '@/api/admin'

describe('useCopilotOAuth', () => {
  it('starts device flow via /admin/copilot/oauth/start', async () => {
    vi.mocked(adminAPI.copilot.startDeviceFlow).mockResolvedValueOnce({
      device_code: 'device-code',
      user_code: 'ABCD-EFGH',
      verification_uri: 'https://github.com/login/device',
      expires_in: 900,
      interval: 5
    })
    const oauth = useCopilotOAuth()

    const ok = await oauth.startDeviceFlow(7)

    expect(ok).toBe(true)
    expect(adminAPI.copilot.startDeviceFlow).toHaveBeenCalledWith({ proxy_id: 7 })
    expect(oauth.deviceCode.value).toBe('device-code')
    expect(oauth.userCode.value).toBe('ABCD-EFGH')
  })

  it('keeps pending responses pollable', async () => {
    const oauth = useCopilotOAuth()
    oauth.deviceCode.value = 'device-code'
    vi.mocked(adminAPI.copilot.pollDeviceFlow).mockResolvedValueOnce({
      status: 'pending',
      message: 'Waiting for GitHub authorization.'
    })

    const result = await oauth.pollDeviceFlow(9)

    expect(result).toBeNull()
    expect(adminAPI.copilot.pollDeviceFlow).toHaveBeenCalledWith({ device_code: 'device-code', proxy_id: 9 })
    expect(oauth.statusMessage.value).toBe('Waiting for GitHub authorization.')
  })

  it('returns credentials and github user info on completion', async () => {
    const oauth = useCopilotOAuth()
    oauth.deviceCode.value = 'device-code'
    vi.mocked(adminAPI.copilot.pollDeviceFlow).mockResolvedValueOnce({
      status: 'complete',
      message: 'done',
      github_login: 'octocat',
      github_name: 'The Octocat',
      github_id: 1,
      credentials: { github_access_token: 'ghu_x', copilot_token: 'cop_x' }
    })

    const result = await oauth.pollDeviceFlow()

    expect(result).toEqual({
      github_login: 'octocat',
      github_name: 'The Octocat',
      github_id: 1,
      credentials: { github_access_token: 'ghu_x', copilot_token: 'cop_x' }
    })
  })
})

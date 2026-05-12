import { describe, expect, it } from 'vitest'
import {
  OPENAI_CC_SWITCH_CODEX_MODEL,
  buildCcSwitchImportDeeplink,
  type CcSwitchClientType
} from '@/utils/ccswitchImport'

function paramsFromDeeplink(deeplink: string): URLSearchParams {
  const query = deeplink.split('?')[1] || ''
  return new URLSearchParams(query)
}

describe('ccswitchImport utils', () => {
  const baseInput = {
    baseUrl: 'https://api.example.com',
    providerName: 'Sub2API',
    apiKey: 'sk-test',
    usageScript: 'return true'
  }

  it('adds the Codex model parameter for Codex imports', () => {
    const params = paramsFromDeeplink(
      buildCcSwitchImportDeeplink({
        ...baseInput,
        clientType: 'codex'
      })
    )

    expect(params.get('resource')).toBe('provider')
    expect(params.get('app')).toBe('codex')
    expect(params.get('endpoint')).toBe(baseInput.baseUrl)
    expect(params.get('model')).toBe(OPENAI_CC_SWITCH_CODEX_MODEL)
    expect(atob(params.get('usageScript') || '')).toBe(baseInput.usageScript)
  })

  it.each([
    { clientType: 'claude', app: 'claude', endpoint: baseInput.baseUrl },
    { clientType: 'gemini', app: 'gemini', endpoint: `${baseInput.baseUrl}/v1beta` },
    { clientType: 'opencode', app: 'opencode', endpoint: baseInput.baseUrl },
    { clientType: 'openclaw', app: 'openclaw', endpoint: baseInput.baseUrl },
    { clientType: 'antigravity', app: 'claude', endpoint: `${baseInput.baseUrl}/antigravity` },
    { clientType: 'copilot', app: 'copilot', endpoint: baseInput.baseUrl }
  ])('does not add a model parameter for $clientType imports', ({ clientType, app, endpoint }) => {
    const params = paramsFromDeeplink(
      buildCcSwitchImportDeeplink({
        ...baseInput,
        clientType: clientType as CcSwitchClientType
      })
    )

    expect(params.get('app')).toBe(app)
    expect(params.get('endpoint')).toBe(endpoint)
    expect(params.has('model')).toBe(false)
  })

  it('preserves legacy platform fallback when no explicit target client is supplied', () => {
    const params = paramsFromDeeplink(
      buildCcSwitchImportDeeplink({
        ...baseInput,
        platform: 'openai'
      })
    )

    expect(params.get('app')).toBe('codex')
    expect(params.get('endpoint')).toBe(baseInput.baseUrl)
    expect(params.get('model')).toBe(OPENAI_CC_SWITCH_CODEX_MODEL)
  })
})

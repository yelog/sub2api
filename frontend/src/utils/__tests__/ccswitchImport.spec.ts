import { describe, expect, it } from 'vitest'
import {
  OPENAI_CC_SWITCH_CODEX_MODEL,
  buildCcSwitchImportDeeplink,
  type CcSwitchClientType
} from '@/utils/ccswitchImport'
import type { GroupPlatform } from '@/types'

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
    { appType: 'opencode' as const, expected: 'opencode' },
    { appType: 'openclaw' as const, expected: 'openclaw' }
  ])('uses explicit $expected CC-Switch app category', ({ appType, expected }) => {
    const params = paramsFromDeeplink(
      buildCcSwitchImportDeeplink({
        ...baseInput,
        platform: 'openai',
        clientType: 'claude',
        appType,
        model: 'gpt-5.5'
      })
    )

    expect(params.get('app')).toBe(expected)
    expect(params.get('model')).toBe('gpt-5.5')
  })

  it('uses the selected model parameter when provided', () => {
    const params = paramsFromDeeplink(
      buildCcSwitchImportDeeplink({
        ...baseInput,
        platform: 'openai',
        clientType: 'claude',
        model: 'gpt-5.5'
      })
    )

    expect(params.get('app')).toBe('codex')
    expect(params.get('model')).toBe('gpt-5.5')
  })

  it.each([
    { platform: 'anthropic' as GroupPlatform, clientType: 'claude' as const, app: 'claude', endpoint: baseInput.baseUrl },
    { platform: 'gemini' as GroupPlatform, clientType: 'gemini' as const, app: 'gemini', endpoint: baseInput.baseUrl },
    { platform: 'openai' as GroupPlatform, clientType: 'opencode' as const, app: 'opencode', endpoint: baseInput.baseUrl },
    { platform: 'openai' as GroupPlatform, clientType: 'openclaw' as const, app: 'openclaw', endpoint: baseInput.baseUrl },
    { platform: 'antigravity' as GroupPlatform, clientType: 'claude' as const, app: 'claude', endpoint: `${baseInput.baseUrl}/antigravity` },
    { platform: 'copilot' as GroupPlatform, clientType: 'copilot' as const, app: 'copilot', endpoint: baseInput.baseUrl }
  ])('does not add a model parameter for $clientType imports', ({ platform, clientType, app, endpoint }) => {
    const params = paramsFromDeeplink(
      buildCcSwitchImportDeeplink({
        ...baseInput,
        platform,
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

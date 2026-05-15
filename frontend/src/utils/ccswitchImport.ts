import type { GroupPlatform } from '@/types'

export const OPENAI_CC_SWITCH_CODEX_MODEL = 'gpt-5.4'

export type CcSwitchClientType = 'claude' | 'codex' | 'gemini' | 'opencode' | 'openclaw' | 'antigravity' | 'copilot'
export type CcSwitchAppType = 'claude' | 'codex' | 'gemini' | 'opencode' | 'openclaw' | 'copilot'

export interface CcSwitchImportConfig {
  app: CcSwitchAppType
  endpoint: string
  model?: string
}

export interface CcSwitchImportDeeplinkInput {
  baseUrl: string
  platform?: GroupPlatform | null
  clientType?: CcSwitchClientType
  appType?: CcSwitchAppType
  providerName: string
  apiKey: string
  usageScript: string
  model?: string | null
}

export function getDefaultCcSwitchAppType(
  platform: GroupPlatform | undefined | null,
  clientType: CcSwitchClientType = 'claude'
): CcSwitchAppType {
  switch (clientType) {
    case 'codex':
      return 'codex'
    case 'gemini':
      return 'gemini'
    case 'opencode':
      return 'opencode'
    case 'openclaw':
      return 'openclaw'
    case 'copilot':
      return 'copilot'
  }

  switch (platform || 'anthropic') {
    case 'openai':
      return 'codex'
    case 'gemini':
      return 'gemini'
    case 'antigravity':
      return 'claude'
    case 'copilot':
      return 'copilot'
    default:
      return 'claude'
  }
}

function ensurePath(baseUrl: string, path: string): string {
  const trimmed = baseUrl.replace(/\/+$/, '')
  return trimmed.endsWith(path) ? trimmed : `${trimmed}${path}`
}

export function resolveCcSwitchImportConfig(
  platform: GroupPlatform | undefined | null,
  clientType: CcSwitchClientType = 'claude',
  baseUrl: string,
  appType: CcSwitchAppType = getDefaultCcSwitchAppType(platform, clientType)
): CcSwitchImportConfig {
  let endpoint = baseUrl
  if (platform === 'antigravity' || clientType === 'antigravity') {
    endpoint = ensurePath(baseUrl, '/antigravity')
  } else if (clientType === 'gemini' && platform !== 'gemini') {
    endpoint = ensurePath(baseUrl, '/v1beta')
  }

  const defaultModel = appType === 'codex' || appType === 'opencode' || appType === 'openclaw'
    ? OPENAI_CC_SWITCH_CODEX_MODEL
    : undefined

  return {
    app: appType,
    endpoint,
    model: defaultModel
  }
}

export function buildCcSwitchImportDeeplink(input: CcSwitchImportDeeplinkInput): string {
  const config = resolveCcSwitchImportConfig(
    input.platform,
    input.clientType,
    input.baseUrl,
    input.appType
  )
  const model = input.model || config.model
  const entries: [string, string][] = [
    ['resource', 'provider'],
    ['app', config.app],
    ['name', input.providerName],
    ['homepage', input.baseUrl],
    ['endpoint', config.endpoint],
    ['apiKey', input.apiKey],
    ['configFormat', 'json'],
    ['usageEnabled', 'true'],
    ['usageScript', btoa(input.usageScript)],
    ['usageAutoInterval', '30']
  ]

  if (model) {
    entries.splice(2, 0, ['model', model])
  }

  return `ccswitch://v1/import?${new URLSearchParams(entries).toString()}`
}

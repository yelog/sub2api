import type { GroupPlatform } from '@/types'

export const OPENAI_CC_SWITCH_CODEX_MODEL = 'gpt-5.4'

export type CcSwitchClientType = 'claude' | 'codex' | 'gemini' | 'opencode' | 'openclaw' | 'antigravity' | 'copilot'

export interface CcSwitchImportConfig {
  app: string
  endpoint: string
  model?: string
}

export interface CcSwitchImportDeeplinkInput {
  baseUrl: string
  platform?: GroupPlatform | null
  clientType?: CcSwitchClientType
  providerName: string
  apiKey: string
  usageScript: string
}

function ensurePath(baseUrl: string, path: string): string {
  const trimmed = baseUrl.replace(/\/+$/, '')
  return trimmed.endsWith(path) ? trimmed : `${trimmed}${path}`
}

export function resolveCcSwitchImportConfig(
  platform: GroupPlatform | undefined | null,
  clientType: CcSwitchClientType | undefined,
  baseUrl: string
): CcSwitchImportConfig {
  switch (clientType) {
    case 'claude':
      return {
        app: 'claude',
        endpoint: baseUrl
      }
    case 'codex':
      return {
        app: 'codex',
        endpoint: baseUrl,
        model: OPENAI_CC_SWITCH_CODEX_MODEL
      }
    case 'gemini':
      return {
        app: 'gemini',
        endpoint: ensurePath(baseUrl, '/v1beta')
      }
    case 'opencode':
      return {
        app: 'opencode',
        endpoint: baseUrl
      }
    case 'openclaw':
      return {
        app: 'openclaw',
        endpoint: baseUrl
      }
    case 'antigravity':
      return {
        app: 'claude',
        endpoint: ensurePath(baseUrl, '/antigravity')
      }
    case 'copilot':
      return {
        app: 'copilot',
        endpoint: baseUrl
      }
  }

  switch (platform || 'anthropic') {
    case 'antigravity':
      return {
        app: clientType === 'gemini' ? 'gemini' : 'claude',
        endpoint: `${baseUrl}/antigravity`
      }
    case 'openai':
      return {
        app: 'codex',
        endpoint: baseUrl,
        model: OPENAI_CC_SWITCH_CODEX_MODEL
      }
    case 'gemini':
      return {
        app: 'gemini',
        endpoint: baseUrl
      }
    default:
      return {
        app: 'claude',
        endpoint: baseUrl
      }
  }
}

export function buildCcSwitchImportDeeplink(input: CcSwitchImportDeeplinkInput): string {
  const config = resolveCcSwitchImportConfig(input.platform, input.clientType, input.baseUrl)
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

  if (config.model) {
    entries.splice(2, 0, ['model', config.model])
  }

  return `ccswitch://v1/import?${new URLSearchParams(entries).toString()}`
}

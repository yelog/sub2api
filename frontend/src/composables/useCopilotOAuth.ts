import { ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores/app'
import { adminAPI } from '@/api/admin'

export interface CopilotOAuthCompleteResult {
  credentials: Record<string, unknown>
  github_login?: string
  github_name?: string
  github_id?: number
}

export function useCopilotOAuth() {
  const appStore = useAppStore()
  const { t } = useI18n()

  const verificationUri = ref('')
  const deviceCode = ref('')
  const userCode = ref('')
  const interval = ref(5)
  const expiresIn = ref(0)
  const loading = ref(false)
  const error = ref('')
  const statusMessage = ref('')

  const authUrl = verificationUri
  const sessionId = deviceCode

  const resetState = () => {
    verificationUri.value = ''
    deviceCode.value = ''
    userCode.value = ''
    interval.value = 5
    expiresIn.value = 0
    loading.value = false
    error.value = ''
    statusMessage.value = ''
  }

  const startDeviceFlow = async (proxyId?: number | null): Promise<boolean> => {
    loading.value = true
    error.value = ''
    statusMessage.value = ''
    verificationUri.value = ''
    deviceCode.value = ''
    userCode.value = ''

    try {
      const payload = proxyId ? { proxy_id: proxyId } : {}
      const response = await adminAPI.copilot.startDeviceFlow(payload)
      verificationUri.value = response.verification_uri
      deviceCode.value = response.device_code
      userCode.value = response.user_code
      interval.value = response.interval
      expiresIn.value = response.expires_in
      return true
    } catch (err: any) {
      error.value = err.response?.data?.detail || t('admin.accounts.oauth.copilot.failedToStartDeviceFlow')
      appStore.showError(error.value)
      return false
    } finally {
      loading.value = false
    }
  }

  const pollDeviceFlow = async (proxyId?: number | null): Promise<CopilotOAuthCompleteResult | null> => {
    if (!deviceCode.value) {
      error.value = t('admin.accounts.oauth.copilot.missingDeviceCode')
      return null
    }
    loading.value = true
    error.value = ''
    try {
      const payload = proxyId ? { device_code: deviceCode.value, proxy_id: proxyId } : { device_code: deviceCode.value }
      const response = await adminAPI.copilot.pollDeviceFlow(payload)
      statusMessage.value = response.message || ''
      if (response.status === 'pending') {
        return null
      }
      return {
        credentials: response.credentials || {},
        github_login: response.github_login,
        github_name: response.github_name,
        github_id: response.github_id
      }
    } catch (err: any) {
      error.value = err.response?.data?.detail || t('admin.accounts.oauth.copilot.failedToPollDeviceFlow')
      appStore.showError(error.value)
      return null
    } finally {
      loading.value = false
    }
  }

  const buildCredentials = (result: CopilotOAuthCompleteResult): Record<string, unknown> => ({ ...(result.credentials || {}) })

  const buildExtraInfo = (result: CopilotOAuthCompleteResult): Record<string, unknown> | undefined => {
    const extra: Record<string, unknown> = {}
    if (result.github_login) extra.github_login = result.github_login
    if (result.github_name) extra.github_name = result.github_name
    if (typeof result.github_id === 'number') extra.github_id = result.github_id
    return Object.keys(extra).length > 0 ? extra : undefined
  }

  return { authUrl, sessionId, verificationUri, deviceCode, userCode, interval, expiresIn, loading, error, statusMessage, resetState, startDeviceFlow, pollDeviceFlow, buildCredentials, buildExtraInfo }
}

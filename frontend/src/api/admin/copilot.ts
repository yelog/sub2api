import { apiClient } from '../client'

export interface CopilotStartDeviceFlowRequest {
  proxy_id?: number
}

export interface CopilotStartDeviceFlowResponse {
  device_code: string
  user_code: string
  verification_uri: string
  expires_in: number
  interval: number
}

export interface CopilotPollDeviceFlowRequest {
  device_code: string
  proxy_id?: number
}

export interface CopilotPollDeviceFlowResponse {
  status: 'pending' | 'complete'
  message?: string
  credentials?: Record<string, unknown>
  github_login?: string
  github_name?: string
  github_id?: number
}

export async function startDeviceFlow(payload: CopilotStartDeviceFlowRequest = {}): Promise<CopilotStartDeviceFlowResponse> {
  const { data } = await apiClient.post<CopilotStartDeviceFlowResponse>('/admin/copilot/oauth/start', payload)
  return data
}

export async function pollDeviceFlow(payload: CopilotPollDeviceFlowRequest): Promise<CopilotPollDeviceFlowResponse> {
  const { data } = await apiClient.post<CopilotPollDeviceFlowResponse>('/admin/copilot/oauth/poll', payload)
  return data
}

export default { startDeviceFlow, pollDeviceFlow }

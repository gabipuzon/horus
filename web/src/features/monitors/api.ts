import { apiRequest } from '../../lib/api'

export type Monitor = {
  id: string
  name: string
  url: string
  interval_seconds: number
  timeout_seconds: number
  expected_status: number
  enabled: boolean
}

export type CreateMonitorInput = Pick<Monitor, 'name' | 'url' | 'interval_seconds' | 'timeout_seconds' | 'expected_status'>

export const monitorApi = {
  list: () => apiRequest<Monitor[]>('/monitors'),
  create: (input: CreateMonitorInput) => apiRequest<Monitor>('/monitors', { method: 'POST', body: JSON.stringify(input) }),
  setEnabled: (id: string, enabled: boolean) => apiRequest<void>(`/monitors/${encodeURIComponent(id)}/${enabled ? 'enable' : 'disable'}`, { method: 'PATCH' }),
  remove: (id: string) => apiRequest<void>(`/monitors/${encodeURIComponent(id)}`, { method: 'DELETE' }),
}

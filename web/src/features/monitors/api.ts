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

export type Check = { id: string; monitor_id: string; status_code: number; latency_ms: number; success: boolean; failure_type: string; error?: string; checked_at: string }
export type CheckSummary = { total_checks: number; successful_checks: number; failed_checks: number; average_latency_ms: number; latest_status: number; uptime_percentage: number | null }
export type Incident = { id: string; monitor_id: string; started_at: string; resolved_at: string | null; is_open: boolean; duration_ms: number; failure_type: string; status_code: number; failure_message?: string }

export const monitorKeys = {
  list: ['monitors'] as const,
  detail: (id: string) => ['monitors', id] as const,
  summary: (id: string) => ['monitors', id, 'summary'] as const,
  checks: (id: string) => ['monitors', id, 'checks', 50, 0] as const,
  incidents: (id: string) => ['monitors', id, 'incidents', 50, 0] as const,
  currentIncident: (id: string) => ['monitors', id, 'incidents', 'current'] as const,
}

export const monitorApi = {
  list: () => apiRequest<Monitor[]>('/monitors'),
  get: (id: string) => apiRequest<Monitor>(`/monitors/${encodeURIComponent(id)}`),
  checks: (id: string) => apiRequest<Check[]>(`/monitors/${encodeURIComponent(id)}/checks?limit=50&offset=0`),
  summary: (id: string) => apiRequest<CheckSummary>(`/monitors/${encodeURIComponent(id)}/summary`),
  incidents: (id: string) => apiRequest<Incident[]>(`/monitors/${encodeURIComponent(id)}/incidents?limit=50&offset=0`),
  currentIncident: async (id: string) => (await apiRequest<Incident | undefined>(`/monitors/${encodeURIComponent(id)}/incidents/current`)) ?? null,
  create: (input: CreateMonitorInput) => apiRequest<Monitor>('/monitors', { method: 'POST', body: JSON.stringify(input) }),
  setEnabled: (id: string, enabled: boolean) => apiRequest<void>(`/monitors/${encodeURIComponent(id)}/${enabled ? 'enable' : 'disable'}`, { method: 'PATCH' }),
  remove: (id: string) => apiRequest<void>(`/monitors/${encodeURIComponent(id)}`, { method: 'DELETE' }),
}

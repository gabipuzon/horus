import { QueryClient } from '@tanstack/react-query'
import { render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { beforeEach, expect, it, vi } from 'vitest'
import { App } from '../app/app'
import type { Check, CheckSummary, Incident, Monitor } from '../features/monitors/api'

const id = 'a0813c85-1d21-49e5-9156-c76680ef867e'
const monitor: Monitor = { id, name: 'Production API', url: 'https://example.com/health', enabled: true, interval_seconds: 30, timeout_seconds: 5, expected_status: 200 }
const summary: CheckSummary = { total_checks: 2, successful_checks: 1, failed_checks: 1, average_latency_ms: 120, latest_status: 503, uptime_percentage: 50 }
const checks: Check[] = [{ id: 'c1', monitor_id: id, success: false, status_code: 503, latency_ms: 240, failure_type: 'http', checked_at: '2026-10-01T05:42:00Z' }, { id: 'c2', monitor_id: id, success: true, status_code: 200, latency_ms: 80, failure_type: '', checked_at: '2026-10-01T05:41:00Z' }]
const incident: Incident = { id: 'i1', monitor_id: id, started_at: '2026-10-01T05:42:00Z', resolved_at: null, is_open: true, duration_ms: 12000, failure_type: 'http', status_code: 503, failure_message: 'unexpected status' }
const json = (body: unknown, status = 200) => new Response(JSON.stringify(body), { status, headers: { 'Content-Type': 'application/json' } })
const empty = () => new Response(null, { status: 204 })

function mockApi(overrides: Record<string, () => Response | Promise<Response>> = {}) {
  const routes: Record<string, () => Response | Promise<Response>> = {
    [`/api/monitors/${id}`]: () => json(monitor),
    [`/api/monitors/${id}/summary`]: () => json(summary),
    [`/api/monitors/${id}/checks?limit=50&offset=0`]: () => json(checks),
    [`/api/monitors/${id}/incidents/current`]: empty,
    [`/api/monitors/${id}/incidents?limit=50&offset=0`]: () => json([]),
    ...overrides,
  }
  vi.stubGlobal('fetch', vi.fn((input: string) => routes[input]?.() ?? Promise.reject(new Error(`Unexpected ${input}`))))
}
function show() {
  window.history.pushState({}, '', `/monitors/${id}`)
  render(<App queryClient={new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } })} />)
}
beforeEach(() => { window.history.pushState({}, '', '/'); vi.unstubAllGlobals() })

it('renders monitor identity, configuration, summary, checks, and chart data', async () => {
  mockApi()
  show()
  expect(await screen.findByRole('heading', { name: 'Production API' })).toBeInTheDocument()
  expect(screen.getByText('● ENABLED')).toBeInTheDocument()
  expect(screen.getByText('Expected HTTP 200 · Every 30 s · Timeout 5 s')).toBeInTheDocument()
  const metrics = within(await screen.findByRole('region', { name: 'Summary' }))
  expect(await metrics.findByText('50.00%')).toBeInTheDocument()
  expect(metrics.getByText('120 ms')).toBeInTheDocument()
  expect(metrics.getByText('HTTP 503')).toBeInTheDocument()
  const table = within(screen.getByRole('region', { name: 'Recent checks' }))
  expect(await table.findByText('● Failed')).toBeInTheDocument()
  expect(table.getByText('Success')).toBeInTheDocument()
  expect(screen.getByRole('img', { name: /highest 240 milliseconds/ })).toBeInTheDocument()
  expect(screen.getByText(/Failed checks with recorded latency are included/)).toBeInTheDocument()
})

it('shows 404 and back route', async () => {
  mockApi({ [`/api/monitors/${id}`]: () => json({ error: 'monitor not found' }, 404) })
  show()
  expect(await screen.findByRole('heading', { name: 'Monitor not found' })).toBeInTheDocument()
  expect(screen.getByRole('link', { name: '← Monitors' })).toHaveAttribute('href', '/monitors')
})

it('shows honest no-check summary and empty checks', async () => {
  mockApi({ [`/api/monitors/${id}/summary`]: () => json({ ...summary, total_checks: 0, latest_status: 0, average_latency_ms: 0, uptime_percentage: null }), [`/api/monitors/${id}/checks?limit=50&offset=0`]: () => json([]) })
  show()
  const metrics = within(await screen.findByRole('region', { name: 'Summary' }))
  expect(await metrics.findAllByText('—')).toHaveLength(3)
  expect(await screen.findByText('No checks recorded yet.')).toBeInTheDocument()
  expect(screen.getByText('Horus will show response history after the first check.')).toBeInTheDocument()
  expect(screen.getByText('No response times recorded yet.')).toBeInTheDocument()
})

it('shows active incident, down state, and incident history', async () => {
  mockApi({ [`/api/monitors/${id}/incidents/current`]: () => json(incident), [`/api/monitors/${id}/incidents?limit=50&offset=0`]: () => json([incident, { ...incident, id: 'i0', is_open: false, resolved_at: '2026-10-01T05:40:00Z' }]) })
  show()
  expect(await screen.findByText('DOWN · Active incident')).toBeInTheDocument()
  expect(screen.getByText('● DOWN')).toBeInTheDocument()
  expect(screen.getByText('unexpected status')).toBeInTheDocument()
  expect(within(screen.getByRole('region', { name: 'Incident history' })).getByText('Resolved')).toBeInTheDocument()
})

it('shows quiet no-current-incident and empty history', async () => {
  mockApi()
  show()
  expect(await screen.findByText('No active incident')).toBeInTheDocument()
  expect(await screen.findByText('No incidents recorded.')).toBeInTheDocument()
  expect(screen.queryByText('HEALTHY')).not.toBeInTheDocument()
})

it('loads sections independently and retries a failed summary', async () => {
  let calls = 0
  mockApi({ [`/api/monitors/${id}/summary`]: () => { calls++; return calls === 1 ? json({ error: 'unavailable' }, 500) : json(summary) } })
  show()
  const metrics = within(await screen.findByRole('region', { name: 'Summary' }))
  expect(await metrics.findByText('Could not load summary.')).toBeInTheDocument()
  expect(await screen.findByText('● Failed')).toBeInTheDocument()
  await userEvent.click(metrics.getByRole('button', { name: 'Retry' }))
  expect(await metrics.findByText('50.00%')).toBeInTheDocument()
})

it('shows section skeletons while requests are pending', async () => {
  mockApi({ [`/api/monitors/${id}/summary`]: () => new Promise(() => {}), [`/api/monitors/${id}/checks?limit=50&offset=0`]: () => new Promise(() => {}), [`/api/monitors/${id}/incidents/current`]: () => new Promise(() => {}), [`/api/monitors/${id}/incidents?limit=50&offset=0`]: () => new Promise(() => {}) })
  show()
  expect(screen.getByRole('status', { name: 'Loading monitor header' })).toBeInTheDocument()
  await screen.findByRole('heading', { name: 'Production API' })
  for (const label of ['summary', 'response time chart', 'recent checks', 'current incident', 'incident history']) expect(screen.getByRole('status', { name: `Loading ${label}` })).toBeInTheDocument()
})

it('toggles configuration locally and confirms deletion before leaving detail', async () => {
  let enabled = true
  const fetchMock = vi.fn(async (path: string, init?: RequestInit) => {
    if (init?.method === 'PATCH') { enabled = false; return empty() }
    if (init?.method === 'DELETE') return empty()
    if (path === `/api/monitors/${id}`) return json({ ...monitor, enabled })
    if (path === '/api/monitors') return json([])
    if (path.endsWith('/summary')) return json(summary)
    if (path.includes('/checks')) return json(checks)
    if (path.endsWith('/incidents/current')) return empty()
    return json([])
  })
  vi.stubGlobal('fetch', fetchMock)
  show()
  await userEvent.click(await screen.findByRole('button', { name: 'Disable' }))
  expect(await screen.findByText('○ DISABLED')).toBeInTheDocument()
  expect(window.location.pathname).toBe(`/monitors/${id}`)
  await userEvent.click(screen.getByRole('button', { name: 'Delete' }))
  expect(screen.getByText(`Delete “${monitor.name}”?`)).toBeInTheDocument()
  expect(fetchMock).not.toHaveBeenCalledWith(`/api/monitors/${id}`, expect.objectContaining({ method: 'DELETE' }))
  await userEvent.click(within(screen.getByRole('dialog')).getByRole('button', { name: 'Delete monitor' }))
  expect(await screen.findByRole('heading', { name: 'Monitors' })).toBeInTheDocument()
  expect(window.location.pathname).toBe('/monitors')
})

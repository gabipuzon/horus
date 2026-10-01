import { QueryClient } from '@tanstack/react-query'
import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { beforeEach, expect, it, vi } from 'vitest'
import { App } from '../app/app'
import type { Monitor } from '../features/monitors/api'

const monitor: Monitor = { id: 'a0813c85-1d21-49e5-9156-c76680ef867e', name: 'Production API', url: 'https://example.com/health', interval_seconds: 30, timeout_seconds: 5, expected_status: 200, enabled: true }
const disabled: Monitor = { ...monitor, id: 'b0813c85-1d21-49e5-9156-c76680ef867e', name: 'Staging', enabled: false }
const json = (body: unknown) => new Response(JSON.stringify(body), { status: 200, headers: { 'Content-Type': 'application/json' } })

function show(path = '/') {
  window.history.pushState({}, '', path)
  render(<App queryClient={new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } })} />)
}

beforeEach(() => vi.unstubAllGlobals())

it('redirects home to Overview and shows an actionable zero-monitor state', async () => {
  vi.stubGlobal('fetch', vi.fn(async () => json([])))
  show()
  expect(await screen.findByRole('heading', { name: 'No monitors yet' })).toBeInTheDocument()
  expect(window.location.pathname).toBe('/overview')
  await userEvent.click(screen.getByRole('button', { name: 'Add monitor' }))
  expect(screen.getByRole('dialog')).toBeInTheDocument()
  expect(within(screen.getByRole('dialog')).getByLabelText('URL')).toBeInTheDocument()
})

it('counts monitor configuration, shows a quiet incident state, and links to detail', async () => {
  const fetchMock = vi.fn(async (path: string) => path === '/api/monitors' ? json([monitor, disabled]) : new Response(null, { status: 204 }))
  vi.stubGlobal('fetch', fetchMock)
  show('/overview')
  const summary = within(await screen.findByRole('region', { name: 'Monitor status summary' }))
  await waitFor(() => expect(summary.getByText('Active incidents')).toBeInTheDocument())
  expect(summary.getByText('Total monitors').nextElementSibling).toHaveTextContent('2')
  expect(summary.getByText('Enabled').nextElementSibling).toHaveTextContent('1')
  expect(summary.getByText('Disabled').nextElementSibling).toHaveTextContent('1')
  expect(await screen.findByText('No active incidents')).toBeInTheDocument()
  expect(screen.getByRole('link', { name: 'Production API' })).toHaveAttribute('href', `/monitors/${monitor.id}`)
  expect(fetchMock).toHaveBeenCalledWith(`/api/monitors/${monitor.id}/incidents/current`, expect.anything())
  expect(fetchMock).toHaveBeenCalledWith(`/api/monitors/${disabled.id}/incidents/current`, expect.anything())
})

it('shows an active problem with incident context', async () => {
  vi.stubGlobal('fetch', vi.fn(async (path: string) => path === '/api/monitors' ? json([monitor]) : json({ id: 'i1', monitor_id: monitor.id, started_at: new Date(Date.now() - 120000).toISOString(), resolved_at: null, is_open: true, duration_ms: 120000, failure_type: 'http', status_code: 503 })))
  show('/overview')
  const problems = within(await screen.findByRole('region', { name: 'Current problems' }))
  expect(await problems.findByText('● DOWN')).toBeInTheDocument()
  expect(problems.getByText('http · HTTP 503')).toBeInTheDocument()
  expect(problems.getByRole('link', { name: 'Production API' })).toHaveAttribute('href', `/monitors/${monitor.id}`)
})

it('checks disabled monitors too because an open incident may outlive disabling', async () => {
  vi.stubGlobal('fetch', vi.fn(async (path: string) => path === '/api/monitors' ? json([disabled]) : json({ id: 'i2', monitor_id: disabled.id, started_at: new Date(Date.now() - 60000).toISOString(), resolved_at: null, is_open: true, duration_ms: 60000, failure_type: 'network', status_code: 0 })))
  show('/overview')
  const problems = within(await screen.findByRole('region', { name: 'Current problems' }))
  expect(await problems.findByRole('link', { name: 'Staging' })).toBeInTheDocument()
  expect(problems.getByText('network')).toBeInTheDocument()
})

it('caps incident requests and labels incomplete coverage', async () => {
  const many = Array.from({ length: 15 }, (_, index) => ({ ...monitor, id: `id-${index}`, name: `Monitor ${index}` }))
  const fetchMock = vi.fn(async (path: string) => path === '/api/monitors' ? json(many) : new Response(null, { status: 204 }))
  vi.stubGlobal('fetch', fetchMock)
  show('/overview')
  expect(await screen.findByText('No active incidents found in checked monitors')).toBeInTheDocument()
  expect(screen.getByText(/Checked 12 of 15 monitors/)).toBeInTheDocument()
  expect(fetchMock.mock.calls.filter(([path]) => String(path).includes('/incidents/current'))).toHaveLength(12)
  expect(screen.queryByText('Active incidents')).not.toBeInTheDocument()
  expect(screen.getByRole('link', { name: 'View all monitors' })).toHaveAttribute('href', '/monitors')
})

it('keeps failed incident checks visible as incomplete coverage with retry', async () => {
  const fetchMock = vi.fn(async (path: string) => path === '/api/monitors' ? json([monitor]) : Promise.reject(new Error('offline')))
  vi.stubGlobal('fetch', fetchMock)
  show('/overview')
  expect(await screen.findByText(/Checked 0 of 1 monitors; 1 incident check failed/)).toBeInTheDocument()
  expect(screen.queryByText('Active incidents')).not.toBeInTheDocument()
  expect(screen.getByRole('button', { name: 'Retry incident checks' })).toBeInTheDocument()
})

it('marks the selected navigation route', async () => {
  vi.stubGlobal('fetch', vi.fn(async () => json([])))
  show('/overview')
  const navigation = within(screen.getByRole('navigation', { name: 'Primary navigation' }))
  expect(navigation.getByRole('link', { name: 'Overview' })).toHaveAttribute('aria-current', 'page')
  await userEvent.click(navigation.getByRole('link', { name: 'Monitors' }))
  expect(navigation.getByRole('link', { name: 'Monitors' })).toHaveAttribute('aria-current', 'page')
})

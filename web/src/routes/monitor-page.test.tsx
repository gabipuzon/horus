import { QueryClient } from '@tanstack/react-query'
import { fireEvent, render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { App } from '../app/app'
import type { Monitor } from '../features/monitors/api'

const production: Monitor = { id: 'a0813c85-1d21-49e5-9156-c76680ef867e', name: 'Production API', url: 'https://example.com/health', interval_seconds: 30, timeout_seconds: 5, expected_status: 200, enabled: true }

function json(body: unknown, status = 200) {
  return new Response(JSON.stringify(body), { status, headers: { 'Content-Type': 'application/json' } })
}

function showApp() {
  window.history.pushState({}, '', '/monitors')
  render(<App queryClient={new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } })} />)
}

beforeEach(() => { window.history.pushState({}, '', '/') })

describe('monitor dashboard', () => {
  it('renders a monitor list using configuration state only', async () => {
    vi.stubGlobal('fetch', vi.fn(async () => json([production])))
    showApp()
    expect(await screen.findByText('Production API')).toBeInTheDocument()
    expect(screen.getByText('ENABLED')).toBeInTheDocument()
    expect(screen.getByText('https://example.com/health')).toBeInTheDocument()
    expect(screen.queryByText('HEALTHY')).not.toBeInTheDocument()
  })

  it('shows skeletons while the list loads', () => {
    vi.stubGlobal('fetch', vi.fn(() => new Promise(() => {})))
    showApp()
    expect(screen.getByRole('status', { name: 'Loading monitors' })).toBeInTheDocument()
  })

  it('shows an actionable empty state', async () => {
    vi.stubGlobal('fetch', vi.fn(async () => json([])))
    showApp()
    expect(await screen.findByText('No monitors yet')).toBeInTheDocument()
    expect(screen.getAllByRole('button', { name: 'Add monitor' })).toHaveLength(2)
    await userEvent.click(screen.getAllByRole('button', { name: 'Add monitor' })[1])
    expect(screen.getByRole('dialog')).toBeInTheDocument()
  })

  it('shows a page error and retries', async () => {
    const fetchMock = vi.fn().mockResolvedValueOnce(json({ error: 'service unavailable' }, 500)).mockResolvedValueOnce(json([production]))
    vi.stubGlobal('fetch', fetchMock)
    showApp()
    expect(await screen.findByText('Could not load monitors.')).toBeInTheDocument()
    await userEvent.click(screen.getByRole('button', { name: 'Retry' }))
    expect(await screen.findByText('Production API')).toBeInTheDocument()
  })

  it('validates create input, sends the typed request, and refreshes the list', async () => {
    let monitors: Monitor[] = []
    const fetchMock = vi.fn(async (_path: string, init?: RequestInit) => {
      if (init?.method === 'POST') { const input = JSON.parse(init.body as string); monitors = [{ ...input, id: production.id, enabled: true }]; return json(monitors[0], 201) }
      return json(monitors)
    })
    vi.stubGlobal('fetch', fetchMock)
    showApp()
    await screen.findByText('No monitors yet')
    await userEvent.click(screen.getAllByRole('button', { name: 'Add monitor' })[0])
    const dialog = screen.getByRole('dialog')
    await userEvent.click(within(dialog).getByRole('button', { name: 'Add monitor' }))
    expect(within(dialog).getByText('Name is required.')).toBeInTheDocument()
    expect(fetchMock).not.toHaveBeenCalledWith('/api/monitors', expect.objectContaining({ method: 'POST' }))
    await userEvent.type(within(dialog).getByLabelText('Name'), 'Production API')
    await userEvent.type(within(dialog).getByLabelText('URL'), 'https://example.com/health')
    await userEvent.click(within(dialog).getByRole('button', { name: 'Add monitor' }))
    expect(await screen.findByText('Production API')).toBeInTheDocument()
    expect(fetchMock).toHaveBeenCalledWith('/api/monitors', expect.objectContaining({ method: 'POST', body: JSON.stringify({ name: 'Production API', url: 'https://example.com/health', interval_seconds: 30, timeout_seconds: 5, expected_status: 200 }) }))
  })

  it('shows backend create errors inside the dialog', async () => {
    vi.stubGlobal('fetch', vi.fn(async (_path: string, init?: RequestInit) => init?.method === 'POST' ? json({ error: 'monitor URL is invalid' }, 400) : json([])))
    showApp()
    await screen.findByText('No monitors yet')
    await userEvent.click(screen.getAllByRole('button', { name: 'Add monitor' })[0])
    const dialog = screen.getByRole('dialog')
    await userEvent.type(within(dialog).getByLabelText('Name'), 'API')
    await userEvent.type(within(dialog).getByLabelText('URL'), 'https://example.com')
    await userEvent.click(within(dialog).getByRole('button', { name: 'Add monitor' }))
    expect(await within(dialog).findByText('monitor URL is invalid')).toBeInTheDocument()
  })

  it('disables and enables a monitor without a confirmation', async () => {
    let enabled = true
    const fetchMock = vi.fn(async (_path: string, init?: RequestInit) => {
      if (init?.method === 'PATCH') { enabled = _path.endsWith('/enable'); return new Response(null, { status: 204 }) }
      return json([{ ...production, enabled }])
    })
    vi.stubGlobal('fetch', fetchMock)
    showApp()
    await screen.findByText('Production API')
    await userEvent.click(screen.getByRole('button', { name: 'Disable Production API' }))
    expect(await screen.findByText('DISABLED')).toBeInTheDocument()
    await userEvent.click(screen.getByRole('button', { name: 'Enable Production API' }))
    await waitFor(() => expect(screen.getByText('ENABLED')).toBeInTheDocument())
    expect(fetchMock).toHaveBeenCalledWith(`/api/monitors/${production.id}/disable`, expect.objectContaining({ method: 'PATCH' }))
    expect(fetchMock).toHaveBeenCalledWith(`/api/monitors/${production.id}/enable`, expect.objectContaining({ method: 'PATCH' }))
  })

  it('requires a named confirmation before deleting and refreshes after success', async () => {
    let monitors = [production]
    const fetchMock = vi.fn(async (_path: string, init?: RequestInit) => {
      if (init?.method === 'DELETE') { monitors = []; return new Response(null, { status: 204 }) }
      return json(monitors)
    })
    vi.stubGlobal('fetch', fetchMock)
    showApp()
    await screen.findByText('Production API')
    await userEvent.click(screen.getByRole('button', { name: 'Delete Production API' }))
    expect(screen.getByText('Delete “Production API”?')).toBeInTheDocument()
    expect(fetchMock).not.toHaveBeenCalledWith(`/api/monitors/${production.id}`, expect.objectContaining({ method: 'DELETE' }))
    await userEvent.click(within(screen.getByRole('dialog')).getByRole('button', { name: 'Delete monitor' }))
    expect(await screen.findByText('No monitors yet')).toBeInTheDocument()
    expect(fetchMock).toHaveBeenCalledWith(`/api/monitors/${production.id}`, expect.objectContaining({ method: 'DELETE' }))
  })

  it('keeps a mutation error beside the affected monitor', async () => {
    vi.stubGlobal('fetch', vi.fn(async (_path: string, init?: RequestInit) => init?.method === 'PATCH' ? json({ error: 'monitor not found' }, 404) : json([production])))
    showApp()
    await screen.findByText('Production API')
    fireEvent.click(screen.getByRole('button', { name: 'Disable Production API' }))
    expect(await screen.findByRole('alert')).toHaveTextContent('monitor not found')
  })
})

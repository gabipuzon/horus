const apiOrigin = (import.meta.env.VITE_HORUS_API_URL ?? '').replace(/\/$/, '')
const apiPrefix = apiOrigin ? '' : '/api'

type APIErrorBody = { error?: string }

export class APIError extends Error {
  constructor(message: string, public readonly status: number) {
    super(message)
    this.name = 'APIError'
  }
}

export async function apiRequest<T>(path: string, init?: RequestInit): Promise<T> {
  let response: Response
  try {
    response = await fetch(`${apiOrigin}${apiPrefix}${path}`, {
      ...init,
      headers: { ...(init?.body ? { 'Content-Type': 'application/json' } : {}), ...init?.headers },
    })
  } catch {
    throw new APIError('Could not connect to Horus. Check that the API is running.', 0)
  }

  if (!response.ok) {
    let message = 'Request failed. Please try again.'
    try {
      const body = (await response.json()) as APIErrorBody
      if (typeof body.error === 'string' && body.error.trim()) message = body.error
    } catch { /* Horus normally returns JSON; keep a safe fallback for proxies. */ }
    throw new APIError(message, response.status)
  }
  if (response.status === 204) return undefined as T
  return response.json() as Promise<T>
}

import { describe, expect, it } from 'vitest'
import { initialValues, validateMonitor } from './validation'

describe('monitor form validation', () => {
  it('accepts a valid monitor', () => {
    expect(validateMonitor({ ...initialValues, name: 'API', url: 'https://example.com/health' })).toEqual({})
  })

  it.each(['not-a-url', 'ftp://example.com', 'https://user:password@example.com', 'https://'])('rejects invalid URL %s', (url) => {
    expect(validateMonitor({ ...initialValues, name: 'API', url })).toHaveProperty('url')
  })

  it('rejects invalid numbers and blank name', () => {
    const errors = validateMonitor({ name: '  ', url: 'https://example.com', interval_seconds: '0', timeout_seconds: '1.5', expected_status: '600' })
    expect(Object.keys(errors)).toEqual(expect.arrayContaining(['name', 'interval_seconds', 'timeout_seconds', 'expected_status']))
  })
})

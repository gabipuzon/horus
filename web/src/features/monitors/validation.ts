import type { CreateMonitorInput } from './api'

export type MonitorFormValues = { name: string; url: string; interval_seconds: string; timeout_seconds: string; expected_status: string }
export type MonitorFormErrors = Partial<Record<keyof MonitorFormValues, string>>

export const initialValues: MonitorFormValues = {
  name: '', url: '', interval_seconds: '30', timeout_seconds: '5', expected_status: '200',
}

function wholeNumber(value: string) {
  return /^\d+$/.test(value) ? Number(value) : NaN
}

export function validateMonitor(values: MonitorFormValues): MonitorFormErrors {
  const errors: MonitorFormErrors = {}
  if (!values.name.trim()) errors.name = 'Name is required.'
  try {
    const url = new URL(values.url)
    if (!['http:', 'https:'].includes(url.protocol) || !url.hostname || url.username || url.password || !/^https?:\/\//i.test(values.url)) {
      errors.url = 'Enter an absolute HTTP or HTTPS URL without credentials.'
    }
  } catch { errors.url = 'Enter an absolute HTTP or HTTPS URL without credentials.' }
  for (const field of ['interval_seconds', 'timeout_seconds'] as const) {
    const number = wholeNumber(values[field])
    if (!Number.isSafeInteger(number) || number < 1 || number > 2147483647) errors[field] = 'Enter a positive whole number of seconds.'
  }
  const status = wholeNumber(values.expected_status)
  if (!Number.isInteger(status) || status < 100 || status > 599) errors.expected_status = 'Enter an HTTP status from 100 to 599.'
  return errors
}

export function toCreateInput(values: MonitorFormValues): CreateMonitorInput {
  return {
    name: values.name.trim(), url: values.url.trim(),
    interval_seconds: Number(values.interval_seconds), timeout_seconds: Number(values.timeout_seconds),
    expected_status: Number(values.expected_status),
  }
}

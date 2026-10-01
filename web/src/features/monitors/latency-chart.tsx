import type { Check } from './api'
import { absoluteTime } from '../../lib/time'

export function LatencyChart({ checks }: { checks: Check[] }) {
  const points = [...checks].filter((check) => Number.isFinite(check.latency_ms) && check.latency_ms >= 0).reverse()
  if (!points.length) return <p className="text-sm text-muted-foreground">No response times recorded yet.</p>
  const max = Math.max(1, ...points.map((point) => point.latency_ms))
  const coordinates = points.map((point, index) => ({ ...point, x: points.length === 1 ? 320 : 36 + index * 568 / (points.length - 1), y: 136 - point.latency_ms / max * 104 }))
  return <div>
    <p className="mb-3 text-xs text-muted-foreground">Response time across {points.length} recent checks, oldest to newest. Failed checks with recorded latency are included. Range: 0–{max} ms.</p>
    <svg role="img" aria-label={`Response time from ${absoluteTime(points[0].checked_at)} to ${absoluteTime(points[points.length - 1].checked_at)}; highest ${max} milliseconds`} viewBox="0 0 640 176" className="h-auto w-full" preserveAspectRatio="xMidYMid meet">
      <line x1="36" y1="136" x2="604" y2="136" stroke="currentColor" className="text-border" />
      <line x1="36" y1="32" x2="604" y2="32" stroke="currentColor" className="text-border" />
      <text x="4" y="35" fill="currentColor" className="text-muted-foreground" fontSize="11">{max}</text>
      <text x="12" y="140" fill="currentColor" className="text-muted-foreground" fontSize="11">0</text>
      <polyline fill="none" stroke="currentColor" strokeWidth="2" className="text-accent" points={coordinates.map((point) => `${point.x},${point.y}`).join(' ')} />
      {coordinates.map((point) => <circle key={point.id} cx={point.x} cy={point.y} r={point.success ? 2.5 : 4} fill="currentColor" className={point.success ? 'text-accent' : 'text-status-down'}><title>{absoluteTime(point.checked_at)}: {point.latency_ms} ms, {point.success ? 'success' : 'failed'}</title></circle>)}
      <text x="36" y="163" fill="currentColor" className="text-muted-foreground" fontSize="11">{absoluteTime(points[0].checked_at)}</text>
      <text x="604" y="163" textAnchor="end" fill="currentColor" className="text-muted-foreground" fontSize="11">{absoluteTime(points[points.length - 1].checked_at)}</text>
    </svg>
  </div>
}

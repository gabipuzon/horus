import type { Check } from './api'
import { absoluteTime } from '../../lib/time'

function compactTime(value: string) {
  return new Intl.DateTimeFormat(undefined, { hour: 'numeric', minute: '2-digit' }).format(new Date(value))
}

export function LatencyChart({ checks }: { checks: Check[] }) {
  const points = [...checks].filter((check) => Number.isFinite(check.latency_ms) && check.latency_ms >= 0).reverse()
  if (!points.length) return <p className="text-sm text-muted-foreground">No response times recorded yet.</p>
  const max = Math.max(1, ...points.map((point) => point.latency_ms))
  const coordinates = points.map((point, index) => ({ ...point, x: points.length === 1 ? 320 : 38 + index * 582 / (points.length - 1), y: 148 - point.latency_ms / max * 120 }))
  return <div>
    <div className="mb-5 flex flex-wrap items-baseline justify-between gap-x-4 gap-y-1 text-xs text-muted-foreground"><p>Last {points.length} checks, oldest to newest</p><p className="tabular-nums">0–{max} ms</p></div>
    <svg role="img" aria-label={`Response time from ${absoluteTime(points[0].checked_at)} to ${absoluteTime(points[points.length - 1].checked_at)}; highest ${max} milliseconds`} viewBox="0 0 640 170" className="h-40 w-full sm:h-52" preserveAspectRatio="none">
      <line x1="38" y1="148" x2="620" y2="148" stroke="currentColor" className="text-border" vectorEffect="non-scaling-stroke" />
      <line x1="38" y1="88" x2="620" y2="88" stroke="currentColor" strokeDasharray="3 5" className="text-border" vectorEffect="non-scaling-stroke" />
      <line x1="38" y1="28" x2="620" y2="28" stroke="currentColor" strokeDasharray="3 5" className="text-border" vectorEffect="non-scaling-stroke" />
      <text x="2" y="31" fill="currentColor" className="text-muted-foreground" fontSize="10">{max}</text>
      <text x="18" y="152" fill="currentColor" className="text-muted-foreground" fontSize="10">0</text>
      <polyline fill="none" stroke="currentColor" strokeWidth="1.25" vectorEffect="non-scaling-stroke" strokeLinejoin="round" strokeLinecap="round" className="text-accent" points={coordinates.map((point) => `${point.x},${point.y}`).join(' ')} />
      {coordinates.map((point) => <circle key={point.id} cx={point.x} cy={point.y} r={point.success ? 2 : 3.5} fill="currentColor" className={point.success ? 'text-accent' : 'text-status-down'}><title>{absoluteTime(point.checked_at)}: {point.latency_ms} ms, {point.success ? 'success' : 'failed'}</title></circle>)}
    </svg>
    <div className="mt-2 flex justify-between gap-4 font-mono text-[10px] leading-4 text-muted-foreground"><time dateTime={points[0].checked_at} className="min-w-0 truncate" title={absoluteTime(points[0].checked_at)}><span className="sm:hidden">{compactTime(points[0].checked_at)}</span><span className="hidden sm:inline">{absoluteTime(points[0].checked_at)}</span></time><time dateTime={points[points.length - 1].checked_at} className="min-w-0 truncate text-right" title={absoluteTime(points[points.length - 1].checked_at)}><span className="sm:hidden">{compactTime(points[points.length - 1].checked_at)}</span><span className="hidden sm:inline">{absoluteTime(points[points.length - 1].checked_at)}</span></time></div>
    <p className="mt-4 text-[11px] leading-4 text-muted-foreground">Failed checks with recorded latency are included.</p>
  </div>
}

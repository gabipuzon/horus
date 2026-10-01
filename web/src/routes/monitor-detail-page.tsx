import { useQuery } from '@tanstack/react-query'
import { Link, useParams } from 'react-router-dom'
import { Button } from '../components/ui/button'
import { Skeleton } from '../components/ui/skeleton'
import { monitorApi, monitorKeys, type Incident } from '../features/monitors/api'
import { LatencyChart } from '../features/monitors/latency-chart'
import { MonitorDetailActions } from '../features/monitors/monitor-detail-actions'
import { APIError } from '../lib/api'
import { absoluteTime, duration, relativeTime } from '../lib/time'

function Section({ title, children }: { title: string; children: React.ReactNode }) {
  return <section aria-label={title} className="rounded-md border border-border bg-surface"><h2 className="border-b border-border px-5 py-3 text-sm font-semibold">{title}</h2><div className="px-5 py-4">{children}</div></section>
}

function ErrorState({ label, retry }: { label: string; retry: () => void }) {
  return <div role="alert" className="flex flex-wrap items-center gap-3 text-sm"><span>Could not load {label}.</span><Button variant="secondary" size="sm" onClick={retry}>Retry</Button></div>
}

function Loading({ label, rows = 2 }: { label: string; rows?: number }) {
  return <div role="status" aria-label={`Loading ${label}`} className="space-y-3">{Array.from({ length: rows }, (_, index) => <Skeleton key={index} className="h-5 w-full max-w-lg" />)}</div>
}

function Timestamp({ value }: { value: string }) {
  return <time dateTime={value} title={absoluteTime(value)} className="tabular-nums">{relativeTime(value)}</time>
}

function IncidentDetails({ incident, active }: { incident: Incident; active: boolean }) {
  const elapsed = active ? Math.max(incident.duration_ms, Date.now() - new Date(incident.started_at).getTime()) : incident.duration_ms
  return <div className="space-y-1 text-sm">
    <p className="font-medium">{active ? 'DOWN · Active incident' : 'Resolved incident'}</p>
    <p className="text-muted-foreground">{incident.failure_type || 'Unknown failure'}{incident.status_code > 0 ? ` · HTTP ${incident.status_code}` : ''}</p>
    {incident.failure_message && <p className="break-words text-muted-foreground">{incident.failure_message}</p>}
    <p className="tabular-nums text-muted-foreground">Started <Timestamp value={incident.started_at} /> · Duration {duration(elapsed)}</p>
  </div>
}

export function MonitorDetailPage() {
  const { id = '' } = useParams()
  const monitor = useQuery({ queryKey: monitorKeys.detail(id), queryFn: () => monitorApi.get(id) })
  const ready = monitor.isSuccess
  const summary = useQuery({ queryKey: monitorKeys.summary(id), queryFn: () => monitorApi.summary(id), enabled: ready })
  const checks = useQuery({ queryKey: monitorKeys.checks(id), queryFn: () => monitorApi.checks(id), enabled: ready })
  const current = useQuery({ queryKey: monitorKeys.currentIncident(id), queryFn: () => monitorApi.currentIncident(id), enabled: ready })
  const incidents = useQuery({ queryKey: monitorKeys.incidents(id), queryFn: () => monitorApi.incidents(id), enabled: ready })

  return <div className="space-y-5">
    <Link to="/monitors" className="inline-flex rounded text-sm text-accent hover:underline focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-accent">← Monitors</Link>
    {monitor.isPending ? <div role="status" aria-label="Loading monitor header" className="space-y-3"><Skeleton className="h-8 w-64" /><Skeleton className="h-5 w-96 max-w-full" /><Skeleton className="h-4 w-80 max-w-full" /></div>
      : monitor.isError ? monitor.error instanceof APIError && monitor.error.status === 404 ? <div role="alert" className="rounded-md border border-border bg-surface p-6"><h1 className="text-xl font-semibold">Monitor not found</h1><p className="mt-2 text-sm text-muted-foreground">This monitor may have been deleted.</p></div>
        : <div role="alert"><h1 className="text-xl font-semibold">Could not load monitor.</h1><Button className="mt-3" variant="secondary" onClick={() => void monitor.refetch()}>Retry</Button></div>
        : <>
          <header className="space-y-2">
            <div className="flex flex-wrap items-center gap-3"><h1 className="text-2xl font-semibold tracking-tight">{monitor.data.name}</h1><span className={`text-xs font-medium ${!monitor.data.enabled ? 'text-muted-foreground' : current.data ? 'text-status-down' : 'text-muted-foreground'}`}>{!monitor.data.enabled ? '○ DISABLED' : current.data ? '● DOWN' : '● ENABLED'}</span></div>
            <p className="break-all font-mono text-xs text-muted-foreground">{monitor.data.url}</p>
            <p className="text-xs tabular-nums text-muted-foreground">Expected HTTP {monitor.data.expected_status} · Every {monitor.data.interval_seconds} s · Timeout {monitor.data.timeout_seconds} s</p>
            <MonitorDetailActions monitor={monitor.data} />
          </header>
          <Section title="Current incident">
            {current.isPending ? <Loading label="current incident" rows={2} /> : current.isError ? <ErrorState label="current incident" retry={() => void current.refetch()} /> : current.data ? <div className="border-l-2 border-status-down pl-4 text-status-down"><IncidentDetails incident={current.data} active /></div> : <p className="text-sm text-muted-foreground">No active incident</p>}
          </Section>
          <Section title="Summary">
            {summary.isPending ? <Loading label="summary" rows={3} /> : summary.isError ? <ErrorState label="summary" retry={() => void summary.refetch()} /> : <><dl className="grid gap-4 sm:grid-cols-4">{[
              ['Uptime', summary.data.uptime_percentage === null ? '—' : `${summary.data.uptime_percentage.toFixed(2)}%`],
              ['Average latency', summary.data.total_checks === 0 ? '—' : `${summary.data.average_latency_ms} ms`],
              ['Latest status', summary.data.latest_status === 0 ? '—' : `HTTP ${summary.data.latest_status}`],
              ['Total checks', String(summary.data.total_checks)],
            ].map(([label, value]) => <div key={label}><dt className="text-xs text-muted-foreground">{label}</dt><dd className="mt-1 text-base font-medium tabular-nums">{value}</dd></div>)}</dl><p className="mt-4 text-xs text-muted-foreground">Uptime is based on recorded checks.</p></>}
          </Section>
          <Section title="Response time">
            {checks.isPending ? <Loading label="response time chart" rows={4} /> : checks.isError ? <ErrorState label="response time" retry={() => void checks.refetch()} /> : <LatencyChart checks={checks.data} />}
          </Section>
          <Section title="Recent checks">
            {checks.isPending ? <Loading label="recent checks" rows={4} /> : checks.isError ? <ErrorState label="recent checks" retry={() => void checks.refetch()} /> : checks.data.length === 0 ? <div className="text-sm"><p>No checks recorded yet.</p><p className="mt-1 text-muted-foreground">Horus will show response history after the first check.</p></div> : <div className="overflow-x-auto"><table className="w-full min-w-[620px] text-left text-sm"><thead className="text-xs text-muted-foreground"><tr><th scope="col" className="pb-2 font-medium">Result</th><th scope="col" className="pb-2 font-medium">HTTP status</th><th scope="col" className="pb-2 font-medium">Latency</th><th scope="col" className="pb-2 font-medium">Failure type</th><th scope="col" className="pb-2 font-medium">Checked at</th></tr></thead><tbody>{checks.data.map((check) => <tr key={check.id} className="border-t border-border/75"><td className={`py-2 ${check.success ? '' : 'text-status-down'}`}>{check.success ? 'Success' : '● Failed'}</td><td className="py-2 tabular-nums">{check.status_code || '—'}</td><td className="py-2 tabular-nums">{check.latency_ms} ms</td><td className="py-2">{check.failure_type || '—'}</td><td className="py-2"><Timestamp value={check.checked_at} /></td></tr>)}</tbody></table></div>}
          </Section>
          <Section title="Incident history">
            {incidents.isPending ? <Loading label="incident history" rows={3} /> : incidents.isError ? <ErrorState label="incident history" retry={() => void incidents.refetch()} /> : incidents.data.length === 0 ? <p className="text-sm text-muted-foreground">No incidents recorded.</p> : <div className="overflow-x-auto"><table className="w-full min-w-[620px] text-left text-sm"><thead className="text-xs text-muted-foreground"><tr><th scope="col" className="pb-2 font-medium">Started</th><th scope="col" className="pb-2 font-medium">State</th><th scope="col" className="pb-2 font-medium">Duration</th><th scope="col" className="pb-2 font-medium">Failure type</th><th scope="col" className="pb-2 font-medium">HTTP status</th></tr></thead><tbody>{incidents.data.map((incident) => <tr key={incident.id} className="border-t border-border/75"><td className="py-2"><Timestamp value={incident.started_at} /></td><td className={`py-2 ${incident.is_open ? 'text-status-down' : 'text-muted-foreground'}`}>{incident.is_open ? 'Open' : 'Resolved'}</td><td className="py-2 tabular-nums">{duration(incident.is_open ? Math.max(incident.duration_ms, Date.now() - new Date(incident.started_at).getTime()) : incident.duration_ms)}</td><td className="py-2">{incident.failure_type}</td><td className="py-2 tabular-nums">{incident.status_code || '—'}</td></tr>)}</tbody></table></div>}
          </Section>
        </>}
  </div>
}

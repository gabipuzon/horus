import { useEffect, useState } from 'react'
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
  return <section aria-label={title}><h2 className="border-b border-border pb-3 text-sm font-semibold">{title}</h2><div className="pt-4">{children}</div></section>
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
  const [checkPage, setCheckPage] = useState(0)
  useEffect(() => setCheckPage(0), [id])
  const monitor = useQuery({ queryKey: monitorKeys.detail(id), queryFn: () => monitorApi.get(id) })
  const ready = monitor.isSuccess
  const summary = useQuery({ queryKey: monitorKeys.summary(id), queryFn: () => monitorApi.summary(id), enabled: ready })
  const checks = useQuery({ queryKey: monitorKeys.checks(id), queryFn: () => monitorApi.checks(id), enabled: ready })
  const recentChecks = useQuery({ queryKey: monitorKeys.checksPage(id, checkPage), queryFn: () => monitorApi.checksPage(id, checkPage), enabled: ready })
  const current = useQuery({ queryKey: monitorKeys.currentIncident(id), queryFn: () => monitorApi.currentIncident(id), enabled: ready })
  const incidents = useQuery({ queryKey: monitorKeys.incidents(id), queryFn: () => monitorApi.incidents(id), enabled: ready })

  return <div className="space-y-8">
    <Link to="/monitors" className="inline-flex rounded text-sm text-muted-foreground hover:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-accent">← Monitors</Link>
    {monitor.isPending ? <div role="status" aria-label="Loading monitor header" className="space-y-3"><Skeleton className="h-8 w-64" /><Skeleton className="h-5 w-96 max-w-full" /><Skeleton className="h-4 w-80 max-w-full" /></div>
      : monitor.isError ? monitor.error instanceof APIError && monitor.error.status === 404 ? <div role="alert" className="rounded-md border border-border bg-surface p-6"><h1 className="text-xl font-semibold">Monitor not found</h1><p className="mt-2 text-sm text-muted-foreground">This monitor may have been deleted.</p></div>
        : <div role="alert"><h1 className="text-xl font-semibold">Could not load monitor.</h1><Button className="mt-3" variant="secondary" onClick={() => void monitor.refetch()}>Retry</Button></div>
        : <>
          <header className="border-b border-border pb-6">
            <div className="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
              <div className="min-w-0">
                <div className="flex flex-wrap items-center gap-x-3 gap-y-2"><h1 className="min-w-0 break-words text-[28px] font-semibold leading-tight tracking-[-0.035em] sm:text-[32px]">{monitor.data.name}</h1><span className={`rounded-full border px-2 py-0.5 text-[11px] font-medium ${!monitor.data.enabled ? 'border-border text-muted-foreground' : current.data ? 'border-status-down/40 text-status-down' : 'border-border text-muted-foreground'}`}>{!monitor.data.enabled ? '○ DISABLED' : current.data ? '● DOWN' : '● ENABLED'}</span></div>
                <p className="mt-2 break-all font-mono text-xs leading-5 text-muted-foreground">{monitor.data.url}</p>
              </div>
              <div className="shrink-0"><MonitorDetailActions monitor={monitor.data} /></div>
            </div>
            <dl className="mt-5 grid grid-cols-3 divide-x divide-border border-t border-border pt-4">{[
              ['Expected status', `HTTP ${monitor.data.expected_status}`],
              ['Check interval', `${monitor.data.interval_seconds} s`],
              ['Timeout', `${monitor.data.timeout_seconds} s`],
            ].map(([label, value]) => <div key={label} className="min-w-0 px-3 first:pl-0 last:pr-0 sm:px-6"><dt className="text-xs text-muted-foreground">{label}</dt><dd className="mt-1.5 text-sm font-medium tabular-nums">{value}</dd></div>)}</dl>
          </header>
          <Section title="Current incident">
            {current.isPending ? <Loading label="current incident" rows={2} /> : current.isError ? <ErrorState label="current incident" retry={() => void current.refetch()} /> : current.data ? <div className="rounded-md border border-status-down/30 bg-status-down/5 p-4"><div className="border-l-2 border-status-down pl-4"><IncidentDetails incident={current.data} active /></div></div> : <p className="text-sm text-muted-foreground">No active incident</p>}
          </Section>
          <Section title="Summary">
            {summary.isPending ? <Loading label="summary" rows={3} /> : summary.isError ? <ErrorState label="summary" retry={() => void summary.refetch()} /> : <><dl className="grid grid-cols-2 overflow-hidden rounded-md border border-border bg-surface sm:grid-cols-4">{[
              ['Uptime', summary.data.uptime_percentage === null ? '—' : `${summary.data.uptime_percentage.toFixed(2)}%`],
              ['Average latency', summary.data.total_checks === 0 ? '—' : `${summary.data.average_latency_ms} ms`],
              ['Latest status', summary.data.latest_status === 0 ? '—' : `HTTP ${summary.data.latest_status}`],
              ['Total checks', String(summary.data.total_checks)],
            ].map(([label, value]) => <div key={label} className="border-r border-b border-border p-4 last:border-r-0 sm:border-b-0"><dt className="text-xs text-muted-foreground">{label}</dt><dd className="mt-2 text-xl font-semibold tabular-nums tracking-tight">{value}</dd></div>)}</dl><p className="mt-3 text-xs text-muted-foreground">Uptime is based on recorded checks.</p></>}
          </Section>
          <div className="grid gap-8 lg:grid-cols-[minmax(0,1.7fr)_minmax(280px,0.8fr)]">
            <div className="min-w-0 space-y-8">
          <Section title="Response time">
            {checks.isPending ? <Loading label="response time chart" rows={4} /> : checks.isError ? <ErrorState label="response time" retry={() => void checks.refetch()} /> : <div className="rounded-md border border-border bg-surface p-4 sm:p-5"><LatencyChart checks={checks.data} /></div>}
          </Section>
          <Section title="Recent checks">
            {recentChecks.isPending ? <Loading label="recent checks" rows={4} /> : recentChecks.isError ? <ErrorState label="recent checks" retry={() => void recentChecks.refetch()} /> : recentChecks.data.length === 0 && checkPage === 0 ? <div className="text-sm"><p>No checks recorded yet.</p><p className="mt-1 text-muted-foreground">Horus will show response history after the first check.</p></div> : <>
              <ul className="divide-y divide-border sm:hidden">{recentChecks.data.slice(0, 10).map((check) => <li key={check.id} className="flex items-start justify-between gap-3 py-3 text-sm"><div><p className={check.success ? '' : 'text-status-down'}>{check.success ? 'Success' : '● Failed'} <span className="ml-1 text-muted-foreground">HTTP {check.status_code || '—'}</span></p>{check.failure_type && <p className="mt-1 text-xs text-muted-foreground">{check.failure_type}</p>}</div><div className="shrink-0 text-right text-xs tabular-nums text-muted-foreground"><p>{check.latency_ms} ms</p><p className="mt-1"><Timestamp value={check.checked_at} /></p></div></li>)}</ul>
              <div className="hidden overflow-x-auto sm:block"><table className="w-full min-w-[620px] text-left text-sm"><thead className="text-xs text-muted-foreground"><tr><th scope="col" className="pb-2 font-medium">Result</th><th scope="col" className="pb-2 font-medium">HTTP status</th><th scope="col" className="pb-2 font-medium">Latency</th><th scope="col" className="pb-2 font-medium">Failure type</th><th scope="col" className="pb-2 font-medium">Checked at</th></tr></thead><tbody>{recentChecks.data.slice(0, 10).map((check) => <tr key={check.id} className="border-t border-border/75"><td className={`py-2 ${check.success ? '' : 'text-status-down'}`}>{check.success ? 'Success' : '● Failed'}</td><td className="py-2 tabular-nums">{check.status_code || '—'}</td><td className="py-2 tabular-nums">{check.latency_ms} ms</td><td className="py-2">{check.failure_type || '—'}</td><td className="py-2"><Timestamp value={check.checked_at} /></td></tr>)}</tbody></table></div>
              <nav aria-label="Recent checks pages" className="mt-4 flex items-center justify-between gap-3 border-t border-border pt-4"><span className="text-xs tabular-nums text-muted-foreground">{recentChecks.data.length ? `${checkPage * 10 + 1}–${checkPage * 10 + Math.min(recentChecks.data.length, 10)}` : 'No checks on this page'}</span><div className="flex gap-2"><Button variant="secondary" size="sm" disabled={checkPage === 0} onClick={() => setCheckPage((page) => Math.max(0, page - 1))}>Previous</Button><Button variant="secondary" size="sm" disabled={recentChecks.data.length <= 10} onClick={() => setCheckPage((page) => page + 1)}>Next</Button></div></nav>
            </>}
          </Section>
            </div>
            <div className="min-w-0">
          <Section title="Incident history">
            {incidents.isPending ? <Loading label="incident history" rows={3} /> : incidents.isError ? <ErrorState label="incident history" retry={() => void incidents.refetch()} /> : incidents.data.length === 0 ? <p className="text-sm text-muted-foreground">No incidents recorded.</p> : <ol className="divide-y divide-border rounded-md border border-border bg-surface px-5">{incidents.data.map((incident) => <li key={incident.id} className="py-4"><div className="flex items-start justify-between gap-2"><span className={`text-sm font-medium ${incident.is_open ? 'text-status-down' : ''}`}>{incident.is_open ? '● Open' : 'Resolved'}</span><span className="text-xs text-muted-foreground"><Timestamp value={incident.started_at} /></span></div><p className="mt-2 break-words text-xs text-muted-foreground">{incident.failure_type || 'Unknown failure'}{incident.status_code > 0 ? ` · HTTP ${incident.status_code}` : ''}</p><p className="mt-1 text-xs tabular-nums text-muted-foreground">Duration {duration(incident.is_open ? Math.max(incident.duration_ms, Date.now() - new Date(incident.started_at).getTime()) : incident.duration_ms)}</p></li>)}</ol>}
          </Section>
            </div>
          </div>
        </>}
  </div>
}

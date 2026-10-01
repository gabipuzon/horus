import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { Link } from 'react-router-dom'
import { Button } from '../components/ui/button'
import { Skeleton } from '../components/ui/skeleton'
import { monitorApi, monitorKeys, type Incident, type Monitor } from '../features/monitors/api'
import { CreateMonitorDialog } from '../features/monitors/create-monitor-dialog'
import { absoluteTime, duration, relativeTime } from '../lib/time'

// The list is newest first. Check only the first 12 monitors, three at a time.
const incidentLimit = 12
const incidentConcurrency = 3
const visibleMonitors = 8

type Problem = { monitor: Monitor; incident: Incident }
type IncidentScan = { problems: Problem[]; failed: number }

async function scanIncidents(monitors: Monitor[]): Promise<IncidentScan> {
  const problems: Problem[] = []
  let failed = 0
  let next = 0
  await Promise.all(Array.from({ length: Math.min(incidentConcurrency, monitors.length) }, async () => {
    while (next < monitors.length) {
      const monitor = monitors[next++]
      try {
        const incident = await monitorApi.currentIncident(monitor.id)
        if (incident) problems.push({ monitor, incident })
      } catch { failed++ }
    }
  }))
  return { problems: problems.sort((a, b) => b.incident.started_at.localeCompare(a.incident.started_at)), failed }
}

export function OverviewPage() {
  const [createOpen, setCreateOpen] = useState(false)
  const [notice, setNotice] = useState('')
  const monitors = useQuery({ queryKey: monitorKeys.list, queryFn: monitorApi.list })
  const enabled = monitors.data?.filter((monitor) => monitor.enabled) ?? []
  const scanned = monitors.data?.slice(0, incidentLimit) ?? []
  const scan = useQuery({
    queryKey: ['overview', 'current-incidents', scanned.map((monitor) => monitor.id)],
    queryFn: () => scanIncidents(scanned),
    enabled: scanned.length > 0,
    staleTime: 60_000,
    refetchOnWindowFocus: false,
  })
  const fullCoverage = monitors.data?.length === 0 || (monitors.data !== undefined && monitors.data.length <= incidentLimit && scan.data?.failed === 0)

  return <div className="space-y-6">
    <header className="flex flex-wrap items-start justify-between gap-4">
      <div><h1 className="text-2xl font-semibold tracking-tight">Overview</h1><p className="mt-2 text-sm text-muted-foreground">Monitor configuration and current problems.</p></div>
      <CreateMonitorDialog open={createOpen} onOpenChange={setCreateOpen} onCreated={setNotice} showTrigger={false} />
    </header>
    {notice && <p role="status" className="text-sm text-accent">{notice}</p>}
    {monitors.isPending ? <div role="status" aria-label="Loading overview" className="space-y-4"><Skeleton className="h-32 w-full" /><Skeleton className="h-24 w-full" /><Skeleton className="h-48 w-full" /></div>
      : monitors.isError ? <div role="alert" className="rounded-md border border-border bg-surface p-5"><h2 className="font-medium">Could not load overview.</h2><Button variant="secondary" className="mt-4" onClick={() => void monitors.refetch()}>Retry</Button></div>
        : monitors.data.length === 0 ? <section className="rounded-md border border-border bg-surface p-6"><h2 className="font-medium">No monitors yet</h2><p className="mt-2 text-sm text-muted-foreground">Create your first monitor to start checking uptime.</p><Button className="mt-5" onClick={() => setCreateOpen(true)}>Add monitor</Button></section>
          : <>
            <section aria-labelledby="problems-heading" className="overflow-hidden rounded-md border border-border bg-surface">
              <div className="border-b border-border px-5 py-3"><h2 id="problems-heading" className="text-sm font-semibold">Current problems</h2></div>
              {scan.isPending ? <div role="status" aria-label="Checking current incidents" className="space-y-3 p-5"><Skeleton className="h-5 w-48" /><Skeleton className="h-5 w-full max-w-md" /></div>
                : scan.isError ? <div role="alert" className="flex flex-wrap items-center gap-3 p-5 text-sm">Could not check current incidents.<Button variant="secondary" size="sm" onClick={() => void scan.refetch()}>Retry</Button></div>
                  : <div className="p-5">
                    {scan.data.problems.length === 0 ? <p className="text-sm text-muted-foreground">{fullCoverage ? 'No active incidents' : 'No active incidents found in checked monitors'}</p>
                      : <ul className="divide-y divide-border">{scan.data.problems.map(({ monitor, incident }) => <li key={monitor.id} className="py-3 first:pt-0 last:pb-0"><div className="flex flex-wrap items-start justify-between gap-2"><div className="min-w-0"><span className="mr-2 text-xs text-status-down">● DOWN</span><Link to={`/monitors/${monitor.id}`} className="rounded font-medium hover:text-accent focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-accent">{monitor.name}</Link><p className="mt-1 break-all font-mono text-xs text-muted-foreground">{monitor.url}</p></div><span className="text-xs tabular-nums text-muted-foreground" title={absoluteTime(incident.started_at)}>Started {relativeTime(incident.started_at)} · {duration(Math.max(incident.duration_ms, Date.now() - new Date(incident.started_at).getTime()))}</span></div><p className="mt-1 text-xs text-muted-foreground">{incident.failure_type || 'Unknown failure'}{incident.status_code > 0 ? ` · HTTP ${incident.status_code}` : ''}</p></li>)}</ul>}
                    {!fullCoverage && <p className="mt-3 text-xs text-muted-foreground">Checked {scanned.length - scan.data.failed} of {monitors.data.length} monitors{scan.data.failed > 0 ? `; ${scan.data.failed} incident ${scan.data.failed === 1 ? 'check' : 'checks'} failed` : ''}. Open a monitor for its current state.</p>}
                    {scan.data.failed > 0 && <Button variant="secondary" size="sm" className="mt-3" onClick={() => void scan.refetch()}>Retry incident checks</Button>}
                  </div>}
            </section>
            <section aria-labelledby="summary-heading" className="rounded-md border border-border bg-surface"><h2 id="summary-heading" className="border-b border-border px-5 py-3 text-sm font-semibold">Monitor status summary</h2><dl className="grid grid-cols-2 gap-4 p-5 sm:grid-cols-3">{[
              ['Total monitors', monitors.data.length], ['Enabled', enabled.length], ['Disabled', monitors.data.length - enabled.length],
              ...(fullCoverage ? [['Active incidents', scan.data?.problems.length ?? 0] as const] : []),
            ].map(([label, count]) => <div key={label}><dt className="text-xs text-muted-foreground">{label}</dt><dd className="mt-1 text-xl font-semibold tabular-nums">{count}</dd></div>)}</dl></section>
            <section aria-labelledby="recent-heading" className="overflow-hidden rounded-md border border-border bg-surface"><div className="flex flex-wrap items-center justify-between gap-2 border-b border-border px-5 py-3"><h2 id="recent-heading" className="text-sm font-semibold">Recent monitors</h2>{monitors.data.length > visibleMonitors && <Link to="/monitors" className="rounded text-xs text-accent hover:underline focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-accent">View all monitors</Link>}</div><div className="overflow-x-auto"><table className="w-full min-w-[530px] text-left text-sm"><thead className="text-xs text-muted-foreground"><tr><th scope="col" className="px-5 py-3 font-medium">State</th><th scope="col" className="px-5 py-3 font-medium">Name</th><th scope="col" className="px-5 py-3 font-medium">URL</th><th scope="col" className="px-5 py-3 text-right font-medium">Interval</th></tr></thead><tbody>{monitors.data.slice(0, visibleMonitors).map((monitor) => <tr key={monitor.id} className="border-t border-border/75"><td className="whitespace-nowrap px-5 py-3 text-xs text-muted-foreground">{monitor.enabled ? '● ENABLED' : '○ DISABLED'}</td><td className="px-5 py-3 font-medium"><Link to={`/monitors/${monitor.id}`} className="rounded hover:text-accent focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-accent">{monitor.name}</Link></td><td className="max-w-72 truncate px-5 py-3 font-mono text-xs text-muted-foreground" title={monitor.url}>{monitor.url}</td><td className="px-5 py-3 text-right tabular-nums">{monitor.interval_seconds} s</td></tr>)}</tbody></table></div></section>
          </>}
  </div>
}

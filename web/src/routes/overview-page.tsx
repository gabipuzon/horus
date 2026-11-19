import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { Link } from 'react-router-dom'
import { Plus } from 'lucide-react'
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

  return <div className="space-y-9">
    <header className="flex flex-wrap items-end justify-between gap-4">
      <div><p className="mb-2 font-mono text-[11px] uppercase tracking-[0.16em] text-accent">Horus / Workspace</p><h1 className="text-[32px] font-semibold tracking-[-0.045em] sm:text-[38px]">Overview</h1><p className="mt-1 text-sm text-muted-foreground">Current incidents and monitor activity.</p></div>
      <Button onClick={() => setCreateOpen(true)}><Plus size={15} /> Add monitor</Button>
      <CreateMonitorDialog open={createOpen} onOpenChange={setCreateOpen} onCreated={setNotice} showTrigger={false} />
    </header>
    {notice && <p role="status" className="text-sm text-accent">{notice}</p>}
    {monitors.isPending ? <div role="status" aria-label="Loading overview" className="space-y-4"><Skeleton className="h-32 w-full" /><Skeleton className="h-24 w-full" /><Skeleton className="h-48 w-full" /></div>
      : monitors.isError ? <div role="alert" className="rounded-md border border-border bg-surface p-6"><h2 className="font-medium">Could not load overview.</h2><p className="mt-1 text-sm text-muted-foreground">Check your connection and try again.</p><Button variant="secondary" className="mt-4" onClick={() => void monitors.refetch()}>Retry</Button></div>
        : monitors.data.length === 0 ? <section className="rounded-md border border-border bg-surface px-6 py-12"><h2 className="text-lg font-medium">No monitors yet</h2><p className="mt-2 text-sm text-muted-foreground">Create your first monitor to start checking uptime.</p><Button variant="secondary" className="mt-5" onClick={() => setCreateOpen(true)}>Add monitor</Button></section>
          : <>
            <div className="border-y border-border lg:grid lg:grid-cols-[minmax(0,1.5fr)_minmax(360px,1fr)]">
              <section aria-labelledby="problems-heading" className="py-6 lg:border-r lg:border-border lg:pr-10">
                <div className="flex items-center justify-between gap-3"><h2 id="problems-heading" className="text-sm font-semibold">Current problems</h2><span className="text-xs text-muted-foreground">Incident scan</span></div>
                {scan.isPending ? <div role="status" aria-label="Checking current incidents" className="mt-6 space-y-3"><Skeleton className="h-10 w-32" /><Skeleton className="h-5 w-full max-w-md" /></div>
                  : scan.isError ? <div role="alert" className="mt-6 flex flex-wrap items-center gap-3 text-sm">Could not check current incidents.<Button variant="secondary" size="sm" onClick={() => void scan.refetch()}>Retry</Button></div>
                    : <div className="mt-4">
                      <div className="flex items-center gap-3"><span className={`text-[42px] font-semibold leading-none tabular-nums tracking-[-0.06em] ${scan.data.problems.length > 0 ? 'text-status-down' : 'text-foreground'}`}>{scan.data.problems.length}</span><p className="text-sm text-muted-foreground">{scan.data.problems.length === 1 ? 'incident found' : 'incidents found'}{!fullCoverage ? ' in checked monitors' : ''}</p></div>
                      {scan.data.problems.length === 0 ? <p className="mt-4 text-sm text-muted-foreground">{fullCoverage ? 'No active incidents' : 'No active incidents found in checked monitors'}</p>
                        : <ul className="mt-4 divide-y divide-border border-t border-border">{scan.data.problems.map(({ monitor, incident }) => <li key={monitor.id} className="py-3 last:pb-0"><div className="flex flex-wrap items-start justify-between gap-3"><div className="min-w-0"><span className="mr-2 text-xs font-medium text-status-down">● DOWN</span><Link to={`/monitors/${monitor.id}`} className="rounded font-medium hover:text-accent focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-accent">{monitor.name}</Link><p className="mt-1 break-all font-mono text-xs text-muted-foreground">{monitor.url}</p></div><span className="text-xs tabular-nums text-muted-foreground" title={absoluteTime(incident.started_at)}>Started {relativeTime(incident.started_at)} · {duration(Math.max(incident.duration_ms, Date.now() - new Date(incident.started_at).getTime()))}</span></div><p className="mt-2 text-xs text-muted-foreground">{incident.failure_type || 'Unknown failure'}{incident.status_code > 0 ? ` · HTTP ${incident.status_code}` : ''}</p></li>)}</ul>}
                      {!fullCoverage && <p className="mt-4 text-xs text-muted-foreground">Checked {scanned.length - scan.data.failed} of {monitors.data.length} monitors{scan.data.failed > 0 ? `; ${scan.data.failed} incident ${scan.data.failed === 1 ? 'check' : 'checks'} failed` : ''}. Open a monitor for its current state.</p>}
                      {scan.data.failed > 0 && <Button variant="secondary" size="sm" className="mt-3" onClick={() => void scan.refetch()}>Retry incident checks</Button>}
                    </div>}
              </section>
              <section aria-labelledby="summary-heading" className="border-t border-border py-6 lg:border-0 lg:pl-10"><h2 id="summary-heading" className="text-sm font-semibold">Monitor status summary</h2><dl className="mt-5 grid grid-cols-2 gap-x-6 gap-y-5">{[
                ['Total monitors', monitors.data.length], ['Enabled', enabled.length], ['Disabled', monitors.data.length - enabled.length],
                ...(fullCoverage ? [['Active incidents', scan.data?.problems.length ?? 0] as const] : []),
              ].map(([label, count]) => <div key={label} className="flex min-w-0 flex-col"><dt className="order-2 mt-2 text-xs text-muted-foreground">{label}</dt><dd className="order-1 text-[25px] font-semibold leading-none tabular-nums tracking-tight">{count}</dd></div>)}</dl><p className="mt-5 text-xs text-muted-foreground">Enabled means checks are scheduled.</p></section>
            </div>
            <section aria-labelledby="recent-heading"><div className="mb-3 flex items-center justify-between gap-2"><h2 id="recent-heading" className="text-lg font-semibold tracking-tight">Recent monitors</h2><Link to="/monitors" className="rounded text-xs text-accent hover:underline focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-accent">View all monitors</Link></div><ul className="divide-y divide-border border-y border-border">{monitors.data.slice(0, visibleMonitors).map((monitor) => <li key={monitor.id} className="grid min-w-0 gap-2 py-4 sm:grid-cols-[minmax(0,1fr)_auto] sm:items-center"><div className="min-w-0"><Link to={`/monitors/${monitor.id}`} className="rounded font-medium hover:text-accent focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-accent">{monitor.name}</Link><p className="mt-1 break-all font-mono text-xs text-muted-foreground">{monitor.url}</p></div><div className="flex items-center gap-4 text-xs text-muted-foreground sm:justify-end"><span>{monitor.enabled ? '● Enabled' : '○ Disabled'}</span><span className="tabular-nums">Every {monitor.interval_seconds} s</span></div></li>)}</ul></section>
          </>}
  </div>
}

import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { RefreshCw } from 'lucide-react'
import { Button } from '../components/ui/button'
import { Skeleton } from '../components/ui/skeleton'
import { monitorApi, monitorKeys } from '../features/monitors/api'
import { CreateMonitorDialog } from '../features/monitors/create-monitor-dialog'
import { MonitorRow } from '../features/monitors/monitor-row'

export function MonitorPage() {
  const [notice, setNotice] = useState('')
  const [createOpen, setCreateOpen] = useState(false)
  const monitors = useQuery({ queryKey: monitorKeys.list, queryFn: monitorApi.list })

  return <>
    <div className="flex flex-wrap items-end justify-between gap-4">
      <div><p className="mb-2 font-mono text-[11px] uppercase tracking-[0.16em] text-accent">Horus / Endpoints</p><h1 className="text-[32px] font-semibold tracking-[-0.045em] sm:text-[38px]">Monitors</h1><p className="mt-1 text-sm text-muted-foreground">Manage the endpoints Horus checks.</p></div>
      <CreateMonitorDialog open={createOpen} onOpenChange={setCreateOpen} onCreated={setNotice} />
    </div>
    {notice && <p role="status" className="mt-5 text-sm text-accent">{notice}</p>}
    <section aria-label="Monitor list" className="mt-9 overflow-hidden rounded-md border border-border bg-surface">
      <div className="flex items-center justify-between border-b border-border px-5 py-3.5">
        <h2 className="text-sm font-semibold">All monitors <span className="ml-1 font-normal tabular-nums text-muted-foreground">{monitors.data ? monitors.data.length : '—'}</span></h2>
        {monitors.data && <Button variant="ghost" size="sm" disabled={monitors.isFetching} onClick={() => void monitors.refetch()} aria-label="Refresh monitors"><RefreshCw size={14} /> Refresh</Button>}
      </div>
      {monitors.isPending ? <div role="status" aria-label="Loading monitors" className="space-y-1 p-4">{[0, 1, 2, 3].map((row) => <div key={row} className="flex items-center gap-6 border-b border-border/50 px-2 py-3 last:border-0"><Skeleton className="w-22" /><Skeleton className="w-36" /><Skeleton className="h-4 max-w-72 flex-1" /><Skeleton className="ml-auto w-24" /></div>)}</div>
        : monitors.isError ? <div role="alert" className="px-5 py-12 text-center"><p className="font-medium">Could not load monitors.</p><Button variant="secondary" className="mt-5" onClick={() => void monitors.refetch()}>Retry</Button></div>
          : monitors.data.length === 0 ? <div className="px-5 py-16 text-center"><p className="font-medium">No monitors yet</p><p className="mt-2 text-sm text-muted-foreground">Create your first monitor to start checking uptime.</p><Button variant="secondary" className="mt-5" onClick={() => setCreateOpen(true)}>Add monitor</Button></div>
            : <ul>{monitors.data.map((monitor) => <MonitorRow key={monitor.id} monitor={monitor} onChanged={setNotice} />)}</ul>}
    </section>
    {monitors.data && monitors.data.length > 0 && <p className="mt-3 text-xs text-muted-foreground">Enabled means checks are scheduled. Current health is not shown in this list.</p>}
  </>
}

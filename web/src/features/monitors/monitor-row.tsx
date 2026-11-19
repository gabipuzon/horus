import { useState } from 'react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { Power, Trash2 } from 'lucide-react'
import { Link } from 'react-router-dom'
import { Button } from '../../components/ui/button'
import { Dialog, DialogContent, DialogDescription, DialogTitle } from '../../components/ui/dialog'
import { monitorApi, type Monitor } from './api'

export function MonitorRow({ monitor, onChanged }: { monitor: Monitor; onChanged: (message: string) => void }) {
  const [confirmDelete, setConfirmDelete] = useState(false)
  const [actionError, setActionError] = useState('')
  const queryClient = useQueryClient()
  const toggle = useMutation({
    mutationFn: () => monitorApi.setEnabled(monitor.id, !monitor.enabled),
    onSuccess: async () => { await queryClient.invalidateQueries({ queryKey: ['monitors'] }); setActionError(''); onChanged(`${monitor.enabled ? 'Disabled' : 'Enabled'} “${monitor.name}”.`) },
    onError: (error) => setActionError(error.message),
  })
  const remove = useMutation({
    mutationFn: () => monitorApi.remove(monitor.id),
    onSuccess: async () => { await queryClient.invalidateQueries({ queryKey: ['monitors'] }); setConfirmDelete(false); setActionError(''); onChanged(`Deleted “${monitor.name}”.`) },
  })

  return <>
    <li className="grid min-w-0 gap-2 border-b border-border/75 px-5 py-4 last:border-0 hover:bg-surface-raised/35 sm:grid-cols-[minmax(0,1fr)_auto] sm:items-center sm:gap-4 sm:px-6">
      <div className="min-w-0">
        <div className="flex flex-wrap items-center gap-x-3 gap-y-1"><Link className="min-w-0 break-words rounded text-base font-medium text-foreground hover:text-accent focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-accent" to={`/monitors/${monitor.id}`}>{monitor.name}</Link><span className="inline-flex items-center gap-1.5 text-[10px] font-medium tracking-wide text-muted-foreground"><span aria-hidden="true" className={`text-sm leading-none ${monitor.enabled ? 'text-muted-foreground' : 'text-status-disabled'}`}>{monitor.enabled ? '●' : '○'}</span>{monitor.enabled ? 'ENABLED' : 'DISABLED'}</span></div>
        <p className="mt-1 break-all font-mono text-xs leading-5 text-muted-foreground" title={monitor.url}>{monitor.url}</p>
        <p className="mt-2 flex flex-wrap gap-x-3 gap-y-1 text-xs tabular-nums text-muted-foreground"><span>Every {monitor.interval_seconds} s</span><span>Timeout {monitor.timeout_seconds} s</span><span>HTTP {monitor.expected_status} expected</span></p>
      </div>
      <div className="sm:border-0">
        <div className="flex items-center gap-1 sm:justify-end">
          <Button variant="ghost" size="sm" disabled={toggle.isPending || remove.isPending} aria-label={`${monitor.enabled ? 'Disable' : 'Enable'} ${monitor.name}`} onClick={() => { setActionError(''); toggle.mutate() }}><Power size={16} /><span>{toggle.isPending ? 'Saving…' : monitor.enabled ? 'Disable' : 'Enable'}</span></Button>
          <Button variant="ghost" size="sm" className="hover:text-status-down" aria-label={`Delete ${monitor.name}`} disabled={toggle.isPending || remove.isPending} onClick={() => { remove.reset(); setConfirmDelete(true) }}><Trash2 size={16} /><span>Delete</span></Button>
        </div>
        {actionError && <p role="alert" className="mt-1 max-w-52 text-xs text-status-down">{actionError}</p>}
      </div>
    </li>
    <Dialog open={confirmDelete} onOpenChange={(next) => { if (!remove.isPending) setConfirmDelete(next) }}>
      <DialogContent aria-describedby={`delete-description-${monitor.id}`}>
        <DialogTitle className="pr-7 text-lg font-semibold">Delete “{monitor.name}”?</DialogTitle>
        <DialogDescription id={`delete-description-${monitor.id}`} className="mt-2 text-sm leading-6 text-muted-foreground">This permanently removes the monitor and its stored checks and incidents.</DialogDescription>
        {remove.isError && <p role="alert" className="mt-4 rounded border border-status-down/40 bg-status-down/10 px-3 py-2 text-sm text-status-down">{remove.error.message}</p>}
        <div className="mt-6 flex justify-end gap-2">
          <Button variant="secondary" disabled={remove.isPending} onClick={() => setConfirmDelete(false)}>Cancel</Button>
          <Button variant="destructive" disabled={remove.isPending} onClick={() => remove.mutate()}>{remove.isPending ? 'Deleting…' : 'Delete monitor'}</Button>
        </div>
      </DialogContent>
    </Dialog>
  </>
}

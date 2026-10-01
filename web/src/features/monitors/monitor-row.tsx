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
    <tr className={`border-b border-border/75 last:border-0 hover:bg-surface-raised/35 ${monitor.enabled ? '' : 'text-muted-foreground'}`}>
      <td className="whitespace-nowrap px-5 py-3.5"><span className="inline-flex items-center gap-2 text-xs font-medium tracking-wide"><span aria-hidden="true" className={`text-base leading-none ${monitor.enabled ? 'text-muted-foreground' : 'text-status-disabled'}`}>{monitor.enabled ? '●' : '○'}</span>{monitor.enabled ? 'ENABLED' : 'DISABLED'}</span></td>
      <td className="max-w-48 truncate px-5 py-3.5 font-medium text-foreground" title={monitor.name}><Link className="rounded text-foreground hover:text-accent focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-accent" to={`/monitors/${monitor.id}`}>{monitor.name}</Link></td>
      <td className="max-w-72 truncate px-5 py-3.5 font-mono text-xs text-muted-foreground" title={monitor.url}>{monitor.url}</td>
      <td className="whitespace-nowrap px-5 py-3.5 text-right tabular-nums">{monitor.interval_seconds} s</td>
      <td className="whitespace-nowrap px-5 py-3.5 text-right tabular-nums">{monitor.timeout_seconds} s</td>
      <td className="px-5 py-3.5 text-right font-mono tabular-nums">{monitor.expected_status}</td>
      <td className="px-5 py-2.5 text-right">
        <div className="flex items-center justify-end gap-1">
          <Button variant="ghost" size="sm" disabled={toggle.isPending || remove.isPending} aria-label={`${monitor.enabled ? 'Disable' : 'Enable'} ${monitor.name}`} onClick={() => { setActionError(''); toggle.mutate() }}><Power size={14} /><span className="hidden xl:inline">{toggle.isPending ? 'Saving…' : monitor.enabled ? 'Disable' : 'Enable'}</span></Button>
          <Button variant="ghost" size="icon" className="hover:text-status-down" aria-label={`Delete ${monitor.name}`} disabled={toggle.isPending || remove.isPending} onClick={() => { remove.reset(); setConfirmDelete(true) }}><Trash2 size={15} /></Button>
        </div>
        {actionError && <p role="alert" className="mt-1 max-w-52 text-xs text-status-down">{actionError}</p>}
      </td>
    </tr>
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

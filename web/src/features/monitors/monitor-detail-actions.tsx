import { useState } from 'react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { useNavigate } from 'react-router-dom'
import { Power, Trash2 } from 'lucide-react'
import { Button } from '../../components/ui/button'
import { Dialog, DialogContent, DialogDescription, DialogTitle } from '../../components/ui/dialog'
import { monitorApi, monitorKeys, type Monitor } from './api'

export function MonitorDetailActions({ monitor }: { monitor: Monitor }) {
  const [confirmDelete, setConfirmDelete] = useState(false)
  const [actionError, setActionError] = useState('')
  const queryClient = useQueryClient()
  const navigate = useNavigate()
  const toggle = useMutation({
    mutationFn: () => monitorApi.setEnabled(monitor.id, !monitor.enabled),
    onSuccess: async () => {
      setActionError('')
      await Promise.all([queryClient.invalidateQueries({ queryKey: monitorKeys.detail(monitor.id) }), queryClient.invalidateQueries({ queryKey: monitorKeys.list })])
    },
    onError: (error) => setActionError(error.message),
  })
  const remove = useMutation({
    mutationFn: () => monitorApi.remove(monitor.id),
    onSuccess: async () => { await queryClient.invalidateQueries({ queryKey: monitorKeys.list }); navigate('/monitors') },
  })
  return <>
    <div className="flex flex-wrap items-center gap-1">
      <Button variant="ghost" size="sm" disabled={toggle.isPending || remove.isPending} onClick={() => { setActionError(''); toggle.mutate() }}><Power size={14} />{toggle.isPending ? 'Saving…' : monitor.enabled ? 'Disable' : 'Enable'}</Button>
      <Button variant="ghost" size="sm" className="hover:text-status-down" disabled={toggle.isPending || remove.isPending} onClick={() => { remove.reset(); setConfirmDelete(true) }}><Trash2 size={14} />Delete</Button>
    </div>
    {actionError && <p role="alert" className="text-xs text-status-down">{actionError}</p>}
    <Dialog open={confirmDelete} onOpenChange={(next) => { if (!remove.isPending) setConfirmDelete(next) }}>
      <DialogContent aria-describedby={`detail-delete-description-${monitor.id}`}>
        <DialogTitle className="pr-7 text-lg font-semibold">Delete “{monitor.name}”?</DialogTitle>
        <DialogDescription id={`detail-delete-description-${monitor.id}`} className="mt-2 text-sm leading-6 text-muted-foreground">This permanently removes the monitor and its stored checks and incidents.</DialogDescription>
        {remove.isError && <p role="alert" className="mt-4 rounded border border-status-down/40 bg-status-down/10 px-3 py-2 text-sm text-status-down">{remove.error.message}</p>}
        <div className="mt-6 flex justify-end gap-2"><Button variant="secondary" disabled={remove.isPending} onClick={() => setConfirmDelete(false)}>Cancel</Button><Button variant="destructive" disabled={remove.isPending} onClick={() => remove.mutate()}>{remove.isPending ? 'Deleting…' : 'Delete monitor'}</Button></div>
      </DialogContent>
    </Dialog>
  </>
}

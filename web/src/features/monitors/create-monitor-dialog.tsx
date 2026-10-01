import { useState, type FormEvent } from 'react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { Plus } from 'lucide-react'
import { Button } from '../../components/ui/button'
import { Dialog, DialogContent, DialogDescription, DialogTitle } from '../../components/ui/dialog'
import { Input } from '../../components/ui/input'
import { Label } from '../../components/ui/label'
import { monitorApi } from './api'
import { initialValues, toCreateInput, validateMonitor, type MonitorFormErrors, type MonitorFormValues } from './validation'

const fields: { key: keyof MonitorFormValues; label: string; hint?: string; inputMode?: 'numeric' }[] = [
  { key: 'name', label: 'Name' },
  { key: 'url', label: 'URL', hint: 'Full HTTP or HTTPS address' },
  { key: 'interval_seconds', label: 'Check interval', hint: 'Seconds between checks', inputMode: 'numeric' },
  { key: 'timeout_seconds', label: 'Timeout', hint: 'Seconds before a check times out', inputMode: 'numeric' },
  { key: 'expected_status', label: 'Expected HTTP status', inputMode: 'numeric' },
]

export function CreateMonitorDialog({ open, onOpenChange, onCreated, showTrigger = true }: { open: boolean; onOpenChange: (open: boolean) => void; onCreated: (message: string) => void; showTrigger?: boolean }) {
  const [values, setValues] = useState<MonitorFormValues>(initialValues)
  const [errors, setErrors] = useState<MonitorFormErrors>({})
  const queryClient = useQueryClient()
  const mutation = useMutation({
    mutationFn: monitorApi.create,
    onSuccess: async (monitor) => {
      await queryClient.invalidateQueries({ queryKey: ['monitors'] })
      onOpenChange(false)
      setValues(initialValues)
      setErrors({})
      onCreated(`Added “${monitor.name}”.`)
    },
  })

  function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    const nextErrors = validateMonitor(values)
    setErrors(nextErrors)
    if (Object.keys(nextErrors).length) return
    mutation.mutate(toCreateInput(values))
  }

  return <>
    {showTrigger && <Button onClick={() => { mutation.reset(); onOpenChange(true) }}><Plus size={16} /> Add monitor</Button>}
    <Dialog open={open} onOpenChange={(next) => { if (!mutation.isPending) onOpenChange(next) }}>
      <DialogContent aria-describedby="create-description">
        <DialogTitle className="text-lg font-semibold">Add monitor</DialogTitle>
        <DialogDescription id="create-description" className="mt-1 text-sm text-muted-foreground">Horus will start checking this URL after creation.</DialogDescription>
        <form onSubmit={submit} noValidate className="mt-6 space-y-4">
          {fields.map(({ key, label, hint, inputMode }) => <div key={key}>
            <Label htmlFor={`monitor-${key}`}>{label}</Label>
            <Input id={`monitor-${key}`} name={key} className="mt-1.5" value={values[key]} inputMode={inputMode} autoComplete="off" aria-invalid={Boolean(errors[key])} aria-describedby={errors[key] ? `${key}-error` : hint ? `${key}-hint` : undefined} onChange={(event) => { setValues({ ...values, [key]: event.target.value }); setErrors({ ...errors, [key]: undefined }); mutation.reset() }} />
            {errors[key] ? <p id={`${key}-error`} role="alert" className="mt-1 text-xs text-status-down">{errors[key]}</p> : hint ? <p id={`${key}-hint`} className="mt-1 text-xs text-muted-foreground">{hint}</p> : null}
          </div>)}
          {mutation.isError && <p role="alert" className="rounded border border-status-down/40 bg-status-down/10 px-3 py-2 text-sm text-status-down">{mutation.error.message}</p>}
          <div className="flex justify-end gap-2 pt-2">
            <Button type="button" variant="secondary" onClick={() => onOpenChange(false)} disabled={mutation.isPending}>Cancel</Button>
            <Button type="submit" disabled={mutation.isPending}>{mutation.isPending ? 'Adding…' : 'Add monitor'}</Button>
          </div>
        </form>
      </DialogContent>
    </Dialog>
  </>
}

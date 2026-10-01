import { cn } from '../../lib/utils'

export function Skeleton({ className }: { className?: string }) {
  return <div aria-hidden="true" className={cn('h-4 rounded bg-surface-raised', className)} />
}

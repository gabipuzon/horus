import * as React from 'react'
import { cn } from '../../lib/utils'

export function Input({ className, ...props }: React.ComponentProps<'input'>) {
  return <input data-slot="input" className={cn('h-10 w-full rounded-md border border-border bg-background px-3 text-sm text-foreground outline-none transition-colors placeholder:text-muted-foreground hover:border-border-strong focus-visible:border-accent focus-visible:ring-2 focus-visible:ring-accent/30 disabled:opacity-50', className)} {...props} />
}

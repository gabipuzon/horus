import * as React from 'react'
import { cva, type VariantProps } from 'class-variance-authority'
import { cn } from '../../lib/utils'

const buttonVariants = cva(
  'inline-flex items-center justify-center gap-2 whitespace-nowrap rounded-md border text-sm font-medium transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-accent focus-visible:ring-offset-2 focus-visible:ring-offset-background disabled:pointer-events-none disabled:opacity-50',
  {
    variants: {
      variant: {
        default: 'border-accent bg-accent px-3.5 py-2 text-background hover:bg-accent/85',
        secondary: 'border-border bg-surface-raised px-3.5 py-2 text-foreground hover:bg-border',
        ghost: 'border-transparent px-3 py-2 text-muted-foreground hover:bg-surface-raised hover:text-foreground',
        destructive: 'border-status-down bg-status-down px-3.5 py-2 text-background hover:bg-status-down/85',
      },
      size: { default: 'h-9', sm: 'h-8', icon: 'h-8 w-8' },
    },
    defaultVariants: { variant: 'default', size: 'default' },
  },
)

type ButtonProps = React.ComponentProps<'button'> & VariantProps<typeof buttonVariants>

export function Button({ className, variant, size, ...props }: ButtonProps) {
  return <button data-slot="button" className={cn(buttonVariants({ variant, size }), className)} {...props} />
}

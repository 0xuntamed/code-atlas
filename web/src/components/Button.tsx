import type { ButtonHTMLAttributes } from 'react'
import { cn } from '../lib/cn'

const base = [
  'inline-flex items-center justify-center gap-2 whitespace-nowrap rounded-control font-semibold',
  'transition-[background-color,border-color,color,transform,box-shadow] duration-150',
  'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 focus-visible:ring-offset-canvas',
  'disabled:pointer-events-none disabled:opacity-45',
].join(' ')

const intentClasses = {
  primary:
    'border border-primary bg-primary text-primary-foreground shadow-[0_12px_30px_oklch(0.79_0.145_173/0.12)] hover:-translate-y-px hover:bg-primary/90',
  secondary:
    'border border-border-strong bg-panel-raised text-foreground hover:border-primary/35 hover:bg-surface',
  ghost:
    'border border-transparent bg-transparent text-muted hover:bg-surface hover:text-foreground',
  outline:
    'border border-border-strong bg-transparent text-muted hover:border-primary/40 hover:bg-primary/5 hover:text-foreground',
  danger: 'border border-danger/50 bg-danger/15 text-danger-foreground hover:bg-danger/25',
} as const

const sizeClasses = {
  sm: 'h-8 px-3 text-xs',
  md: 'h-10 px-4 text-sm',
  lg: 'h-11 px-5 text-sm',
  icon: 'size-9 p-0',
} as const

type ButtonProps = ButtonHTMLAttributes<HTMLButtonElement> & {
  intent?: keyof typeof intentClasses
  size?: keyof typeof sizeClasses
}

export function Button({
  className,
  intent = 'primary',
  size = 'md',
  type = 'button',
  ...props
}: ButtonProps) {
  return (
    <button
      className={cn(base, intentClasses[intent], sizeClasses[size], className)}
      type={type}
      {...props}
    />
  )
}

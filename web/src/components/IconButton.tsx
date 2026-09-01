import { ComponentChildren, JSX } from 'preact'

export interface IconButtonProps extends JSX.HTMLAttributes<HTMLButtonElement> {
  icon: ComponentChildren
  'aria-label': string
  variant?: 'ghost' | 'secondary' | 'danger'
  size?: 'sm' | 'md'
  disabled?: boolean
}

export function IconButton({
  icon,
  'aria-label': ariaLabel,
  variant = 'ghost',
  size = 'md',
  className = '',
  disabled = false,
  ...props
}: IconButtonProps) {
  const variantClass = `btn-${variant}`
  const sizeClass = size === 'sm' ? 'btn-sm' : ''

  return (
    <button
      className={`btn btn-icon ${variantClass} ${sizeClass} ${className}`.trim()}
      aria-label={ariaLabel}
      title={ariaLabel}
      disabled={disabled}
      {...props}
    >
      {icon}
    </button>
  )
}

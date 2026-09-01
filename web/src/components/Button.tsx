import { ComponentChildren, JSX } from 'preact'

export interface ButtonProps extends JSX.HTMLAttributes<HTMLButtonElement> {
  variant?: 'primary' | 'secondary' | 'danger' | 'ghost'
  size?: 'sm' | 'md'
  icon?: ComponentChildren
  children?: ComponentChildren
  disabled?: boolean
  loading?: boolean
}

export function Button({
  variant = 'secondary',
  size = 'md',
  icon,
  children,
  className = '',
  disabled = false,
  loading = false,
  ...props
}: ButtonProps) {
  const variantClass = `btn-${variant}`
  const sizeClass = size === 'sm' ? 'btn-sm' : ''

  return (
    <button
      className={`btn ${variantClass} ${sizeClass} ${className}`.trim()}
      disabled={disabled || loading}
      {...props}
    >
      {loading ? (
        <span className="btn-spinner" />
      ) : (
        icon && <span className="btn-icon-wrapper">{icon}</span>
      )}
      {children && <span>{children}</span>}
    </button>
  )
}

import React, { ReactNode } from 'react'

type StatusType = string

interface BadgeProps {
  status?: StatusType
  label?: string
  className?: string
  size?: 'sm' | 'md' | 'lg'
  children?: ReactNode
}

const statusConfig: Record<StatusType, { bg: string; text: string; dot: string }> = {
  active: {
    bg: 'bg-success-500/20',
    text: 'text-success-300',
    dot: 'bg-success-500',
  },
  inactive: {
    bg: 'bg-neutral-700/20',
    text: 'text-neutral-400',
    dot: 'bg-neutral-500',
  },
  error: {
    bg: 'bg-error-500/20',
    text: 'text-error-300',
    dot: 'bg-error-500',
  },
  pending: {
    bg: 'bg-warning-500/20',
    text: 'text-warning-300',
    dot: 'bg-warning-500',
  },
  warning: {
    bg: 'bg-warning-500/20',
    text: 'text-warning-300',
    dot: 'bg-warning-500',
  },
}

const sizeClasses = {
  sm: 'px-2 py-1 text-xs',
  md: 'px-3 py-1.5 text-sm',
  lg: 'px-4 py-2 text-base',
}

export const Badge: React.FC<BadgeProps> = ({
  status,
  label,
  className = '',
  size = 'md',
  children,
}) => {
  if (children && className) {
    return <span className={`inline-flex items-center gap-2 ${sizeClasses[size]} rounded font-medium ${className}`}>{children}</span>
  }

  if (!status) return null

  const config = statusConfig[status]
  const statusLabel = label || status.charAt(0).toUpperCase() + status.slice(1)

  return (
    <span
      className={`inline-flex items-center gap-2 ${sizeClasses[size]} rounded-full font-medium ${config.bg} ${config.text} ${className}`}
    >
      <span className={`w-2 h-2 rounded-full ${config.dot} animate-pulse`} />
      {statusLabel}
    </span>
  )
}

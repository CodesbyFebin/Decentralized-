import React, { ReactNode } from 'react'

interface CardProps {
  children: ReactNode
  variant?: 'default' | 'glass' | 'outlined'
  className?: string
  onClick?: () => void
  hoverable?: boolean
}

export const Card: React.FC<CardProps> = ({
  children,
  variant = 'default',
  className = '',
  onClick,
  hoverable = false,
}) => {
  let baseClass =
    'rounded-lg border border-neutral-700 transition-all duration-200'

  const variantClasses = {
    default: 'bg-neutral-900 border-neutral-700',
    glass:
      'bg-gradient-to-br from-neutral-800/50 to-neutral-900/50 backdrop-blur-md border border-primary-500/20 shadow-lg shadow-primary-500/10',
    outlined: 'bg-transparent border-2 border-neutral-700',
  }

  const hoverClass = hoverable
    ? 'hover:border-primary-500 hover:shadow-lg hover:shadow-primary-500/20 cursor-pointer'
    : ''

  return (
    <div
      className={`${baseClass} ${variantClasses[variant]} ${hoverClass} ${className}`}
      onClick={onClick}
    >
      {children}
    </div>
  )
}

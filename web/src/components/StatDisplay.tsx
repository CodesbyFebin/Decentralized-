import React from 'react'
import { Card } from './Card'

interface StatDisplayProps {
  label: string
  value: string | number
  unit?: string
  trend?: {
    direction: 'up' | 'down' | 'neutral'
    percent: number
  }
  icon?: React.ReactNode
  className?: string
}

export const StatDisplay: React.FC<StatDisplayProps> = ({
  label,
  value,
  unit = '',
  trend,
  icon,
  className = '',
}) => {
  const trendColors = {
    up: 'text-success-500',
    down: 'text-error-500',
    neutral: 'text-neutral-400',
  }

  const trendIcons = {
    up: '↑',
    down: '↓',
    neutral: '→',
  }

  return (
    <Card variant="glass" className={className}>
      <div className="p-6">
        <div className="flex items-start justify-between">
          <div className="flex-1">
            <p className="text-neutral-400 text-sm font-medium">{label}</p>
            <div className="mt-2 flex items-baseline gap-2">
              <span className="text-3xl font-bold text-primary-500">{value}</span>
              {unit && <span className="text-neutral-500 text-sm">{unit}</span>}
            </div>
            {trend && (
              <div className={`mt-2 text-sm font-medium ${trendColors[trend.direction]}`}>
                {trendIcons[trend.direction]} {Math.abs(trend.percent)}% vs last period
              </div>
            )}
          </div>
          {icon && <div className="text-primary-500/60">{icon}</div>}
        </div>
      </div>
    </Card>
  )
}

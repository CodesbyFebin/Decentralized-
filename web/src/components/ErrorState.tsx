import React from 'react'
import { Button } from './Button'

interface ErrorStateProps {
  title: string
  message: string
  onRetry?: () => void
  fullPage?: boolean
}

export const ErrorState: React.FC<ErrorStateProps> = ({
  title,
  message,
  onRetry,
  fullPage = false,
}) => {
  const content = (
    <div className="flex flex-col items-center justify-center">
      <div className="w-16 h-16 rounded-full bg-error-500/20 flex items-center justify-center mb-4">
        <span className="text-3xl">⚠️</span>
      </div>
      <h3 className="text-xl font-semibold text-white mb-2">{title}</h3>
      <p className="text-neutral-400 text-center max-w-md mb-6">{message}</p>
      {onRetry && (
        <Button variant="primary" size="md" onClick={onRetry}>
          Try Again
        </Button>
      )}
    </div>
  )

  if (fullPage) {
    return (
      <div className="flex items-center justify-center min-h-screen bg-neutral-950">
        {content}
      </div>
    )
  }

  return <div className="py-12">{content}</div>
}

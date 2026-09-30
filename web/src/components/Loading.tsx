import React from 'react'

interface LoadingProps {
  size?: 'sm' | 'md' | 'lg'
  fullPage?: boolean
  message?: string
}

export const Loading: React.FC<LoadingProps> = ({
  size = 'md',
  fullPage = false,
  message = 'Loading...',
}) => {
  const sizeClasses = {
    sm: 'w-6 h-6',
    md: 'w-12 h-12',
    lg: 'w-16 h-16',
  }

  const spinner = (
    <div className={`${sizeClasses[size]} animate-spin`}>
      <svg
        className="w-full h-full text-primary-500"
        xmlns="http://www.w3.org/2000/svg"
        fill="none"
        viewBox="0 0 24 24"
      >
        <circle
          className="opacity-25"
          cx="12"
          cy="12"
          r="10"
          stroke="currentColor"
          strokeWidth="4"
        />
        <path
          className="opacity-75"
          fill="currentColor"
          d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
        />
      </svg>
    </div>
  )

  if (fullPage) {
    return (
      <div className="flex items-center justify-center min-h-screen bg-neutral-950">
        <div className="text-center">
          {spinner}
          {message && <p className="mt-4 text-neutral-400">{message}</p>}
        </div>
      </div>
    )
  }

  return (
    <div className="flex flex-col items-center justify-center py-12">
      {spinner}
      {message && <p className="mt-4 text-neutral-400">{message}</p>}
    </div>
  )
}

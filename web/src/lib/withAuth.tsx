import React, { useEffect } from 'react'
import { useRouter } from 'next/router'
import { Loading } from '@/components/Loading'

export function withAuth<P extends object>(
  Component: React.ComponentType<P>
): React.ComponentType<P> {
  return function ProtectedComponent(props: P) {
    const router = useRouter()
    const [isLoading, setIsLoading] = React.useState(true)

    useEffect(() => {
      const token = localStorage.getItem('auth_token')
      if (!token) {
        router.push('/login')
      } else {
        setIsLoading(false)
      }
    }, [router])

    if (isLoading) {
      return <Loading message="Loading..." />
    }

    return <Component {...props} />
  }
}

import type { AppProps } from 'next/app'
import { useEffect } from 'react'
import '@/styles/globals.css'

export default function App({ Component, pageProps }: AppProps) {
  useEffect(() => {
    // Initialize app
    const root = document.documentElement
    root.setAttribute('data-theme', 'dark')
  }, [])

  return <Component {...pageProps} />
}

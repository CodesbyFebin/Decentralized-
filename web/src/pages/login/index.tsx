import React, { useState } from 'react'
import { useRouter } from 'next/router'
import { apiClient } from '@/lib/api'

const LoginPage: React.FC = () => {
  const router = useRouter()
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(false)

  const handleLogin = async (e: React.FormEvent) => {
    e.preventDefault()
    setError(null)
    setLoading(true)

    try {
      const response = await apiClient.post('/auth/login', {
        email,
        password,
      })

      if (response.success && response.data?.token) {
        localStorage.setItem('auth_token', response.data.token)
        router.push('/dashboard')
      } else {
        setError(response.error || 'Login failed')
      }
    } catch (err: any) {
      setError(err.response?.data?.error || 'An error occurred during login')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="min-h-screen bg-neutral-950 flex items-center justify-center p-4">
      <div className="w-full max-w-md">
        <div className="text-center mb-8">
          <h1 className="text-3xl font-bold text-primary-500 mb-2">Decentralized.Host</h1>
          <p className="text-neutral-400">Sovereign Infrastructure Management</p>
        </div>

        <div className="bg-neutral-900 border border-neutral-700 rounded-lg p-8">
          <h2 className="text-xl font-bold text-white mb-6">Sign In</h2>

          {error && (
            <div className="mb-4 p-3 bg-error-500/20 border border-error-500/50 rounded-lg text-error-300 text-sm">
              {error}
            </div>
          )}

          <form onSubmit={handleLogin} className="space-y-4">
            <div>
              <label className="block text-sm font-medium text-neutral-300 mb-2">
                Email Address
              </label>
              <input
                type="email"
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                placeholder="you@example.com"
                className="w-full px-4 py-2 bg-neutral-800 border border-neutral-700 rounded-lg text-white placeholder-neutral-500 focus:outline-none focus:border-primary-500 focus:ring-1 focus:ring-primary-500"
                required
                disabled={loading}
              />
            </div>

            <div>
              <label className="block text-sm font-medium text-neutral-300 mb-2">
                Password
              </label>
              <input
                type="password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                placeholder="••••••••"
                className="w-full px-4 py-2 bg-neutral-800 border border-neutral-700 rounded-lg text-white placeholder-neutral-500 focus:outline-none focus:border-primary-500 focus:ring-1 focus:ring-primary-500"
                required
                disabled={loading}
              />
            </div>

            <button
              type="submit"
              disabled={loading}
              className="w-full mt-6 px-4 py-2 bg-primary-500 hover:bg-primary-600 disabled:bg-neutral-700 text-white font-medium rounded-lg transition-colors"
            >
              {loading ? 'Signing in...' : 'Sign In'}
            </button>
          </form>

          <div className="mt-6 text-center">
            <p className="text-neutral-400 text-sm">
              Don't have an account?{' '}
              <button
                onClick={() => router.push('/register')}
                className="text-primary-500 hover:text-primary-400 font-medium"
              >
                Sign up here
              </button>
            </p>
          </div>
        </div>

        <div className="mt-8 text-center text-neutral-500 text-sm">
          <p>© 2024 Decentralized.Host. All rights reserved.</p>
        </div>
      </div>
    </div>
  )
}

export default LoginPage

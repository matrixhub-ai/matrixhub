import { CurrentUser, type GetCurrentUserResponse } from '@matrixhub/api-ts/v1alpha1/current_user.pb'
import {
  useEffect,
  useMemo,
  useState,
} from 'react'

import { AuthContext, type AuthContextValue } from '@/context/auth'

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const [user, setUser] = useState<GetCurrentUserResponse | null>(null)
  const [loading, setLoading] = useState(true)

  const fetchUser = async () => {
    try {
      const resp = await CurrentUser.GetCurrentUser({})

      setUser(resp)
    } catch (error) {
      console.error('Failed to fetch current user:', error)
      setUser(null)
    } finally {
      setLoading(false)
    }
  }

  const contextValue: AuthContextValue = useMemo(() => ({
    user,
    loading,
    refetch: () => {
      setLoading(() => true)
      fetchUser()
    },
  }), [user, loading])

  useEffect(() => {
    fetchUser()
  }, [])

  return (
    <AuthContext value={contextValue}>
      {children}
    </AuthContext>
  )
}

import { CurrentUser } from '@matrixhub/api-ts/v1alpha1/current_user.pb'
import {
  useEffect,
  useMemo,
  useState,
} from 'react'

import { ProjectRolesContext, type ProjectRolesContextValue } from '@/context/projectRoles'

export function ProjectRolesProvider({ children }: { children: React.ReactNode }) {
  const [roles, setRoles] = useState<ProjectRolesContextValue['roles'] | null>(null)
  const [loading, setLoading] = useState(true)

  const fetchRoles = async () => {
    try {
      const resp = await CurrentUser.GetProjectRoles({})

      setRoles(resp.projectRoles || {})
    } catch (error) {
      console.error('Failed to fetch current project roles:', error)
      setRoles(null)
    } finally {
      setLoading(false)
    }
  }

  const contextValue: ProjectRolesContextValue = useMemo(() => ({
    roles,
    loading,
    refetch: () => {
      setLoading(() => true)
      fetchRoles()
    },
  }), [roles, loading])

  useEffect(() => {
    fetchRoles()
  }, [])

  return (
    <ProjectRolesContext value={contextValue}>
      {children}
    </ProjectRolesContext>
  )
}

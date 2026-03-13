import { Projects } from '@matrixhub/api-ts/v1alpha1/project.pb'
import { useEffect, useState } from 'react'

const PAGE_SIZE = 1000

interface SelectOption {
  value: string
  label: string
}

export function useProjectOptions(currentProjectId: string) {
  const [options, setOptions] = useState<SelectOption[]>(() => (
    currentProjectId
      ? [{
          value: currentProjectId,
          label: currentProjectId,
        }]
      : []
  ))
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    let cancelled = false

    async function fetchProjects() {
      setLoading(true)

      try {
        const res = await Projects.ListProjects({
          page: 1,
          pageSize: PAGE_SIZE,
        })
        const nextOptions = new Map<string, SelectOption>()

        if (currentProjectId) {
          nextOptions.set(currentProjectId, {
            value: currentProjectId,
            label: currentProjectId,
          })
        }

        for (const project of res.projects ?? []) {
          if (!project.name) {
            continue
          }

          nextOptions.set(project.name, {
            value: project.name,
            label: project.name,
          })
        }

        if (!cancelled) {
          setOptions(Array.from(nextOptions.values()))
        }
      } catch {
        if (!cancelled) {
          setOptions(currentProjectId
            ? [{
                value: currentProjectId,
                label: currentProjectId,
              }]
            : [])
        }
      } finally {
        if (!cancelled) {
          setLoading(false)
        }
      }
    }

    fetchProjects()

    return () => {
      cancelled = true
    }
  }, [currentProjectId])

  return {
    options,
    loading,
  }
}

import { getRouteApi } from '@tanstack/react-router'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'

import {
  useModelLibraryLabels,
  useModelProjects,
  useModelTaskLabels,
} from '@/features/models/models.query'
import { LibraryFilterPanel } from '@/shared/components/resource-filter-panel/LibraryFilterPanel'
import { ProjectFilterPanel } from '@/shared/components/resource-filter-panel/ProjectFilterPanel'
import { ResourceFilterPanels as SharedResourceFilterPanel } from '@/shared/components/resource-filter-panel/ResourceFilterPanels'
import { TaskFilterPanel } from '@/shared/components/resource-filter-panel/TaskFilterPanel'

import type { ModelsCatalogSearch } from '@/routes/(auth)/(app)/models'
import type { FilterTabDefinition } from '@/shared/components/resource-filter-panel/types.ts'

const modelsRouteApi = getRouteApi('/(auth)/(app)/models/')

type ModelFilterSearch = Pick<ModelsCatalogSearch, 'task' | 'library' | 'project'>

type ModelFilterTab = keyof ModelFilterSearch

function getModelSearchState(search: ModelFilterSearch) {
  const modelFilterTabs: ModelFilterTab[] = ['task', 'library', 'project']

  for (const tab of modelFilterTabs) {
    const value = search[tab]

    if (typeof value === 'string') {
      return {
        queryTab: tab,
        queryValue: value.split(','),
      }
    }
  }

  return {
    queryTab: modelFilterTabs[0],
    queryValue: [],
  }
}

export function ModelsFilterPanel() {
  const { t } = useTranslation()
  const navigate = modelsRouteApi.useNavigate()

  const {
    data: taskLabels = [],
    isLoading: taskLabelsLoading,
  } = useModelTaskLabels()

  const {
    data: libraryLabels = [],
    isLoading: libraryLabelsLoading,
  } = useModelLibraryLabels()

  const {
    data: projects = [],
    isLoading: projectsLoading,
  } = useModelProjects()

  const search = modelsRouteApi.useSearch()
  const {
    queryTab, queryValue,
  } = getModelSearchState(search)

  // optimistic state for better UX when switching tabs, will be overridden by url state if different
  const [activeTab, setActiveTab] = useState<ModelFilterTab>(queryTab)
  const [selectedNames, setSelectedNames] = useState<string[]>(queryValue)

  const getSelectedNamesForTab = (tab: ModelFilterTab) => {
    return queryTab === tab ? selectedNames : []
  }

  const handleSelectLabels = (name: string) => {
    const labels = (queryTab && activeTab === queryTab)
      ? selectedNames
      : []

    const newVal = labels.includes(name)
      ? labels.filter(val => val !== name)
      : [...labels, name]

    setSelectedNames(newVal)

    navigate({
      search: {
        ...search,
        task: undefined,
        library: undefined,
        project: undefined,
        page: 1,
        [activeTab]: newVal.filter(Boolean).length ? newVal.filter(Boolean).join(',') : undefined,
      },
    })
  }

  const handleSelectProject = (name: string) => {
    const newVal = (queryTab && activeTab === queryTab)
      ? (selectedNames.includes(name) ? undefined : name)
      : name

    setSelectedNames(newVal ? [newVal] : [])

    navigate({
      search: {
        ...search,
        task: undefined,
        library: undefined,
        project: undefined,
        page: 1,
        [activeTab]: newVal,
      },
    })
  }

  const tabs: FilterTabDefinition[] = [
    {
      label: t('model.list.task'),
      value: 'task',
      panel: (
        <TaskFilterPanel
          options={taskLabels}
          loading={taskLabelsLoading}
          searchPlaceholder={t('model.list.placeholder.task')}
          selectedNames={getSelectedNamesForTab('task')}
          onSelect={handleSelectLabels}
        />
      ),
    },
    {
      label: t('model.list.library'),
      value: 'library',
      panel: (
        <LibraryFilterPanel
          options={libraryLabels}
          loading={libraryLabelsLoading}
          searchPlaceholder={t('model.list.placeholder.library')}
          selectedNames={getSelectedNamesForTab('library')}
          onSelect={handleSelectLabels}
        />
      ),
    },
    {
      label: t('model.list.project'),
      value: 'project',
      panel: (
        <ProjectFilterPanel
          options={projects}
          loading={projectsLoading}
          searchPlaceholder={t('model.list.placeholder.project')}
          selectedNames={getSelectedNamesForTab('project')}
          onSelect={handleSelectProject}
        />
      ),
    },
  ]

  return (
    <SharedResourceFilterPanel
      tabs={tabs}
      activeTab={activeTab}
      onTabChange={val => setActiveTab(val as ModelFilterTab)}
    />
  )
}

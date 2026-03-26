import {
  type GetModelTreeRequest, type ListModelCommitsRequest, Models,
} from '@matrixhub/api-ts/v1alpha1/model.pb'
import { Projects } from '@matrixhub/api-ts/v1alpha1/project.pb'
import {
  keepPreviousData, queryOptions, useQuery,
} from '@tanstack/react-query'

import { MODEL_COMMITS_PAGE_SIZE } from '@/utils/constants.ts'

import type { ModelsCatalogSearch } from '@/routes/(auth)/(app)/models'

export const PAGE_SIZE = 6

export type ModelsSortField = 'updatedAt'

export interface ModelsSearch {
  q: string
  sort: ModelsSortField
  order: 'asc' | 'desc'
  page: number
}

interface ModelListQueryKeyParams {
  q: string
  sort: string | undefined
  page: number
}

interface ModelCatalogListQueryKeyParams extends ModelListQueryKeyParams {
  task: string | undefined
  library: string | undefined
  project: string | undefined
}

// -- Query key factory --

export const modelKeys = {
  all: ['models'] as const,
  lists: () => [...modelKeys.all, 'list'] as const,
  list: (projectId: string, params: ModelListQueryKeyParams) => [...modelKeys.lists(), projectId, params] as const,
  catalogList: (params: ModelCatalogListQueryKeyParams) => [...modelKeys.lists(), params] as const,
  taskLabels: () => [...modelKeys.all, 'task-labels'] as const,
  libraryLabels: () => [...modelKeys.all, 'library-labels'] as const,
  projects: () => [...modelKeys.all, 'projects'] as const,
  commits: () => [...modelKeys.all, 'commit-list'] as const,
  commitDetails: () => [...modelKeys.all, 'commit-detail'] as const,
  details: () => [...modelKeys.all, 'detail'] as const,
  detail: (projectId: string, modelName: string) => [...modelKeys.details(), projectId, modelName] as const,
  commitsList: (projectId: string, modelName: string, params: Pick<ListModelCommitsRequest, 'revision' | 'page'>) => [
    ...modelKeys.commits(), projectId, modelName, params,
  ] as const,
  commitDetail: (projectId: string, modelName: string, commitId: string) => [
    ...modelKeys.commitDetails(),
    projectId,
    modelName,
    commitId,
  ] as const,
}

export const modelRevisionKeys = {
  all: ['modelRevisions'] as const,
  detail: (projectId: string, modelName: string) => [...modelRevisionKeys.all, projectId, modelName] as const,
}

export const modelTreeKeys = {
  all: ['modelTree'] as const,
  detail: (
    projectId: string,
    modelName: string,
    params?: Pick<GetModelTreeRequest, 'revision' | 'path'>,
  ) => [...modelTreeKeys.all, projectId, modelName, params] as const,
}

// -- Query options factory --

export function projectModelsQueryOptions(projectId: string, search: ModelsSearch) {
  const sortParam = toSortParam(search.sort, search.order)

  return queryOptions({
    queryKey: modelKeys.list(projectId, {
      q: search.q,
      sort: sortParam,
      page: search.page,
    }),
    queryFn: () => Models.ListModels({
      project: projectId,
      search: search.q || undefined,
      sort: sortParam,
      page: search.page,
      pageSize: PAGE_SIZE,
    }),
  })
}

export function modelsCatalogQueryOptions(search: ModelsCatalogSearch) {
  const sortParam = toSortParam(search.sort, search.order)
  const page = search.page ?? 1

  return queryOptions({
    queryKey: modelKeys.catalogList({
      q: search.q || '',
      sort: sortParam,
      page,
      task: search.task,
      library: search.library,
      project: search.project,
    }),
    queryFn: () => Models.ListModels({
      labels: splitFilterCsv(search.task ?? search.library),
      project: search.project,
      search: search.q || undefined,
      sort: sortParam,
      page,
      pageSize: PAGE_SIZE,
    }),
  })
}

export function modelTaskLabelsQueryOptions() {
  return queryOptions({
    queryKey: modelKeys.taskLabels(),
    queryFn: async () => {
      const response = await Models.ListModelTaskLabels({})

      return response.items ?? []
    },
  })
}

export function modelLibraryLabelsQueryOptions() {
  return queryOptions({
    queryKey: modelKeys.libraryLabels(),
    queryFn: async () => {
      const response = await Models.ListModelFrameLabels({})

      return response.items ?? []
    },
  })
}

export function modelProjectsQueryOptions() {
  return queryOptions({
    queryKey: modelKeys.projects(),
    queryFn: async () => {
      const response = await Projects.ListProjects({
        pageSize: -1,
      })

      return response.projects ?? []
    },
  })
}

export function modelQueryOptions(projectId: string, modelName: string) {
  return queryOptions({
    queryKey: modelKeys.detail(projectId, modelName),
    queryFn: () => Models.GetModel({
      project: projectId,
      name: modelName,
    }),
  })
}

export function modelRevisionsQueryOptions(projectId: string, modelName: string) {
  return queryOptions({
    queryKey: modelRevisionKeys.detail(projectId, modelName),
    queryFn: () => Models.ListModelRevisions({
      project: projectId,
      name: modelName,
    }),
  })
}

export function modelTreeQueryOptions(
  projectId: string,
  modelName: string,
  params?: Pick<GetModelTreeRequest, 'revision' | 'path'>,
) {
  return queryOptions({
    queryKey: modelTreeKeys.detail(projectId, modelName, params),
    queryFn: () => Models.GetModelTree({
      project: projectId,
      name: modelName,
      ...params,
    }),
  })
}

export function modelCommitsQueryOptions(
  projectId: string,
  modelName: string,
  params: Pick<ListModelCommitsRequest, 'revision' | 'page'>,
) {
  return queryOptions({
    queryKey: modelKeys.commitsList(projectId, modelName, params),
    queryFn: () => Models.ListModelCommits({
      project: projectId,
      name: modelName,
      revision: params.revision,
      page: params.page,
      pageSize: MODEL_COMMITS_PAGE_SIZE,
    }),
  })
}

export function modelCommitQueryOptions(
  projectId: string,
  modelName: string,
  commitId: string,
) {
  return queryOptions({
    queryKey: modelKeys.commitDetail(projectId, modelName, commitId),
    queryFn: () => Models.GetModelCommit({
      project: projectId,
      name: modelName,
      id: commitId,
    }),
  })
}

// -- Custom hook --

export function useModels(projectId: string, search: ModelsSearch) {
  return useQuery({
    ...projectModelsQueryOptions(projectId, search),
    placeholderData: keepPreviousData,
  })
}

export function useModelTree(
  projectId: string,
  modelName: string,
  params?: Pick<GetModelTreeRequest, 'revision' | 'path'>,
) {
  return useQuery({
    ...modelTreeQueryOptions(projectId, modelName, params),
  })
}

export function useModelCommits(
  projectId: string,
  modelName: string,
  params: Pick<ListModelCommitsRequest, 'revision' | 'page'>,
) {
  return useQuery({
    ...modelCommitsQueryOptions(projectId, modelName, params),
    placeholderData: keepPreviousData,
  })
}

// -- Internal helpers --

function toSortParam(field?: ModelsSearch['sort'], order?: ModelsSearch['order']) {
  return field === 'updatedAt' && order === 'asc'
    ? 'updated_at_asc'
    : 'updated_at_desc'
}

function splitFilterCsv(value: string | undefined) {
  if (!value) {
    return undefined
  }

  const items = value
    .split(',')
    .map(item => item.trim())
    .filter(Boolean)

  return items.length > 0 ? items : undefined
}

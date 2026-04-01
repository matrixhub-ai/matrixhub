import {
  type GetModelBlobRequest,
  type GetModelTreeRequest,
  type ListModelCommitsRequest,
  type ListModelsRequest,
  Models,
} from '@matrixhub/api-ts/v1alpha1/model.pb'
import { Projects } from '@matrixhub/api-ts/v1alpha1/project.pb'
import { queryOptions, useQuery } from '@tanstack/react-query'

import { DEFAULT_PAGE_SIZE } from '@/utils/constants.ts'

import type { ModelsCatalogSearch } from '@/routes/(auth)/(app)/models'
import type { ProjectModelsSearch } from '@/routes/(auth)/(app)/projects/$projectId/models'

// -- Query key factory --

export const modelKeys = {
  all: ['models'] as const,

  lists: () => [...modelKeys.all, 'list'] as const,
  listByProject: (projectId: string, params: ListModelsRequest) => [...modelKeys.lists(), projectId, params] as const,
  listByCategory: (params: ListModelsRequest) => [...modelKeys.lists(), params] as const,

  details: () => [...modelKeys.all, 'detail'] as const,
  detail: (projectId: string, modelName: string) => [...modelKeys.details(), projectId, modelName] as const,

  taskLabels: () => [...modelKeys.all, 'task-labels'] as const,
  libraryLabels: () => [...modelKeys.all, 'library-labels'] as const,
  projects: () => [...modelKeys.all, 'projects'] as const,

  commits: () => [...modelKeys.all, 'commit-list'] as const,
  commitsList: (projectId: string, modelName: string, params: Pick<ListModelCommitsRequest, 'revision' | 'page' | 'pageSize'>) => [
    ...modelKeys.commits(), projectId, modelName, params,
  ] as const,

  commitDetails: () => [...modelKeys.all, 'commit-detail'] as const,
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

export const modelBlobKeys = {
  all: ['modelBlob'] as const,
  detail: (
    projectId: string,
    modelName: string,
    params?: Pick<GetModelBlobRequest, 'revision' | 'path'>,
  ) => [...modelBlobKeys.all, projectId, modelName, params] as const,
}

// -- Query options factory --

export function projectModelsQueryOptions(projectId: string, search: ProjectModelsSearch) {
  const sortParam = toSortParam(search.sort, search.order)

  return queryOptions({
    queryKey: modelKeys.listByProject(projectId, {
      search: search.q,
      sort: sortParam,
      page: search.page,
    }),
    queryFn: () => Models.ListModels({
      project: projectId,
      search: search.q || undefined,
      sort: sortParam,
      page: search.page,
      pageSize: DEFAULT_PAGE_SIZE,
    }),
  })
}

export function catalogModelsQueryOptions(
  search: ModelsCatalogSearch & {
    popular?: boolean
    pageSize?: number
  }) {
  const sortParam = toSortParam(search.sort, search.order)

  return queryOptions({
    queryKey: modelKeys.listByCategory({
      search: search.q,
      sort: search.popular ? undefined : sortParam,
      page: search.page,
      project: search.project,
      labels: splitFilterCsv(search.task ?? search.library),
      popular: search?.popular,
    }),
    queryFn: () => Models.ListModels({
      search: search.popular ? undefined : search.q,
      sort: search.popular ? undefined : sortParam,
      project: search.project,
      labels: splitFilterCsv(search.task ?? search.library),
      page: search.page,
      pageSize: search?.pageSize ?? DEFAULT_PAGE_SIZE,
      popular: search?.popular,
    }),
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

export function modelBlobQueryOptions(
  projectId: string,
  modelName: string,
  params?: Pick<GetModelBlobRequest, 'revision' | 'path'>,
) {
  return queryOptions({
    queryKey: modelBlobKeys.detail(projectId, modelName, params),
    queryFn: () => Models.GetModelBlob({
      project: projectId,
      name: modelName,
      ...params,
    }),
  })
}

export function modelCommitsQueryOptions(
  projectId: string,
  modelName: string,
  params: Pick<ListModelCommitsRequest, 'revision' | 'page' | 'pageSize'>,
) {
  const normalizedParams = {
    ...params,
    pageSize: params.pageSize ?? DEFAULT_PAGE_SIZE,
  }

  return queryOptions({
    queryKey: modelKeys.commitsList(projectId, modelName, normalizedParams),
    queryFn: () => Models.ListModelCommits({
      project: projectId,
      name: modelName,
      revision: normalizedParams.revision,
      page: normalizedParams.page,
      pageSize: normalizedParams.pageSize,
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

export function useModelTree(
  projectId: string,
  modelName: string,
  params?: Pick<GetModelTreeRequest, 'revision' | 'path'>,
) {
  return useQuery({
    ...modelTreeQueryOptions(projectId, modelName, params),
  })
}

export function useModelBlob(
  projectId: string,
  modelName: string,
  params?: Pick<GetModelBlobRequest, 'revision' | 'path'>,
) {
  return useQuery({
    ...modelBlobQueryOptions(projectId, modelName, params),
  })
}

export function useModelTaskLabels() {
  return useQuery({
    queryKey: modelKeys.taskLabels(),
    queryFn: async () => {
      const response = await Models.ListModelTaskLabels({})

      return response.items ?? []
    },
  })
}

export function useModelLibraryLabels() {
  return useQuery({
    queryKey: modelKeys.libraryLabels(),
    queryFn: async () => {
      const response = await Models.ListModelFrameLabels({})

      return response.items ?? []
    },
  })
}

export function useModelProjects() {
  return useQuery({
    queryKey: modelKeys.projects(),
    queryFn: async () => {
      const response = await Projects.ListProjects({
        page: 1,
        pageSize: -1,
      })

      return response.projects ?? []
    },
  })
}

// -- Internal helpers --

export function toSortParam(field?: ModelsCatalogSearch['sort'], order?: ModelsCatalogSearch['order']) {
  if (!field || !order) {
    return undefined
  }

  return field === 'updatedAt' && order === 'asc'
    ? 'updated_at_asc'
    : 'updated_at_desc'
}

export function splitFilterCsv(value: string | undefined) {
  if (!value) {
    return undefined
  }

  const items = value
    .split(',')
    .map(item => item.trim())
    .filter(Boolean)

  return items.length > 0 ? items : undefined
}

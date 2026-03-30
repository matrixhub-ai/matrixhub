import {
  type GetModelBlobRequest,
  type GetModelTreeRequest,
  type ListModelCommitsRequest,
  type ListModelsRequest,
  Models,
} from '@matrixhub/api-ts/v1alpha1/model.pb'
import { Projects } from '@matrixhub/api-ts/v1alpha1/project.pb'
import {
  keepPreviousData, queryOptions, useQuery,
} from '@tanstack/react-query'

import { DEFAULT_PAGE_SIZE } from '@/utils/constants.ts'

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

// -- Query key factory --

export const modelKeys = {
  all: ['models'] as const,
  lists: () => [...modelKeys.all, 'list'] as const,
  list: (projectId: string, params: ModelListQueryKeyParams) => [...modelKeys.lists(), projectId, params] as const,
  catalogList: (params: ListModelsRequest) => [...modelKeys.lists(), params] as const,
  taskLabels: () => [...modelKeys.all, 'task-labels'] as const,
  libraryLabels: () => [...modelKeys.all, 'library-labels'] as const,
  projects: () => [...modelKeys.all, 'projects'] as const,
  commits: () => [...modelKeys.all, 'commit-list'] as const,
  commitDetails: () => [...modelKeys.all, 'commit-detail'] as const,
  details: () => [...modelKeys.all, 'detail'] as const,
  detail: (projectId: string, modelName: string) => [...modelKeys.details(), projectId, modelName] as const,
  commitsList: (projectId: string, modelName: string, params: Pick<ListModelCommitsRequest, 'revision' | 'page' | 'pageSize'>) => [
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

export const modelBlobKeys = {
  all: ['modelBlob'] as const,
  detail: (
    projectId: string,
    modelName: string,
    params?: Pick<GetModelBlobRequest, 'revision' | 'path'>,
  ) => [...modelBlobKeys.all, projectId, modelName, params] as const,
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
      pageSize: DEFAULT_PAGE_SIZE,
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

export function useModels(params: ListModelsRequest) {
  return useQuery({
    queryKey: modelKeys.catalogList(params),
    queryFn: () => Models.ListModels({
      pageSize: DEFAULT_PAGE_SIZE,
      ...params,
    }),
    placeholderData: keepPreviousData,
  })
}

export function useProjectModels(projectId: string, search: ModelsSearch) {
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

export function useModelBlob(
  projectId: string,
  modelName: string,
  params?: Pick<GetModelBlobRequest, 'revision' | 'path'>,
) {
  return useQuery({
    ...modelBlobQueryOptions(projectId, modelName, params),
  })
}

export function useModelCommits(
  projectId: string,
  modelName: string,
  params: Pick<ListModelCommitsRequest, 'revision' | 'page' | 'pageSize'>,
) {
  return useQuery({
    ...modelCommitsQueryOptions(projectId, modelName, params),
    placeholderData: keepPreviousData,
  })
}

export function useModelCommit(
  projectId: string,
  modelName: string,
  commitId: string,
) {
  return useQuery({
    ...modelCommitQueryOptions(projectId, modelName, commitId),
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

export function toSortParam(field?: ModelsSearch['sort'], order?: ModelsSearch['order']) {
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

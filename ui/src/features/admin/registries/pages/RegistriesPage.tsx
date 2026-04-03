import { useQueryClient } from '@tanstack/react-query'
import { getRouteApi } from '@tanstack/react-router'

import { useRouteListState } from '@/shared/hooks/useRouteListState'

import { CreateRegistryAction } from '../components/CreateRegistryAction'
import { RegistriesTable } from '../components/RegistriesTable'
import {
  adminRegistryKeys,
  useRegistries,
} from '../registries.query'
import { DEFAULT_REGISTRIES_PAGE } from '../registries.schema'
import { getRegistryRowId } from '../registries.utils'

const registriesRouteApi = getRouteApi('/(auth)/admin/registries')

export function RegistriesPage() {
  const queryClient = useQueryClient()
  const navigate = registriesRouteApi.useNavigate()
  const search = registriesRouteApi.useSearch()
  const {
    data,
    isLoading,
    isFetching,
  } = useRegistries(search)
  const registries = data?.registries ?? []
  const pagination = data?.pagination

  const refreshRegistries = () => queryClient.invalidateQueries({
    queryKey: adminRegistryKeys.lists(),
  })

  const {
    onSearchChange,
    onRefresh,
    onPageChange,
  } = useRouteListState({
    search,
    navigate,
    records: registries,
    getRecordId: getRegistryRowId,
    refresh: refreshRegistries,
  })

  return (
    <RegistriesTable
      data={registries}
      pagination={pagination}
      loading={isLoading}
      fetching={isFetching}
      page={search.page ?? DEFAULT_REGISTRIES_PAGE}
      searchValue={search.query ?? ''}
      onSearchChange={onSearchChange}
      onRefresh={onRefresh}
      onPageChange={onPageChange}
      toolbarExtra={<CreateRegistryAction />}
    />
  )
}

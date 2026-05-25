import { notifications } from '@mantine/notifications'
import {
  MutationCache,
  QueryCache,
  QueryClient,
} from '@tanstack/react-query'

import i18n from '@/i18n'

import type { NotificationMeta } from '@/types/tanstack-query'

export function getErrorMessage(error: unknown): string {
  // Try to extract a user-friendly error message from the error object if possible.
  // This is a simple heuristic and can be improved based on the error shapes used in the project.
  if (error instanceof Error || (typeof error === 'object' && error !== null && 'message' in error)) {
    return (error as { message: string }).message
  }

  return String(error)
}

export const queryClient = new QueryClient({
  queryCache: new QueryCache({
    onError: (error, query) => {
      // Only notify for background refetch failures
      // (queries that already had data in cache)
      const meta = query.meta as NotificationMeta | undefined

      if (query.state.data !== undefined && !meta?.skipNotification) {
        notifications.show({
          title: i18n.t('common.backgroundRefreshFailed'),
          message: getErrorMessage(error),
          color: 'red',
        })
      }
    },
  }),
  mutationCache: new MutationCache({
    onSuccess: (_data, _variables, _context, mutation) => {
      const meta = mutation.meta as NotificationMeta | undefined

      if (meta?.skipNotification) {
        return
      }

      if (meta?.successMessage) {
        notifications.show({
          message: meta.successMessage,
          color: 'green',
        })
      }

      // Auto-invalidate related queries
      if (meta?.invalidates) {
        for (const key of meta.invalidates) {
          queryClient.invalidateQueries({ queryKey: key })
        }
      }
    },
    onError: (error, _variables, _context, mutation) => {
      const meta = mutation.meta as NotificationMeta | undefined

      if (meta?.skipNotification) {
        return
      }

      notifications.show({
        title: meta?.errorMessage ?? i18n.t('common.operationFailed'),
        message: getErrorMessage(error),
        color: 'red',
      })
    },
  }),
  defaultOptions: {
    queries: {
      staleTime: 30_000,
      gcTime: 5 * 60_000,
      refetchOnWindowFocus: false,
      retry: 0,
    },
  },
})

i18n.on('languageChanged', () => {
  void queryClient.invalidateQueries({
    predicate: query => Boolean(
      (query.meta as NotificationMeta | undefined)?.localeDependent,
    ),
  })
})

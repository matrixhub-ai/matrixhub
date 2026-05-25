import {
  Anchor,
  Box,
  Group,
  Text,
} from '@mantine/core'
import { useDisclosure } from '@mantine/hooks'
import {
  CurrentUser, AccessTokenStatus, type AccessToken,
} from '@matrixhub/api-ts/v1alpha1/current_user.pb'
import { useMutation } from '@tanstack/react-query'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'

import { profileKeys } from '@/features/profile/profile.query'
import { formatExpiredAt } from '@/features/profile/profile.utils.ts'
import { DataTable, type DataTableRowActionsProps } from '@/shared/components/DataTable'
import { ModalWrapper } from '@/shared/components/ModalWrapper'
import { formatDateTime } from '@/shared/utils/date'

import type {
  MRT_ColumnDef,
  MRT_Row,
} from 'mantine-react-table'

interface AccessTokenTableProps {
  tokens: AccessToken[]
}

const StatusCell = ({ row }: { row: MRT_Row<AccessToken> }) => {
  const { t } = useTranslation()

  const statusColor = {
    [AccessTokenStatus.ACCESS_TOKEN_STATUS_VALID]: 'teal.6',
    [AccessTokenStatus.ACCESS_TOKEN_STATUS_EXPIRED]: 'red.6',
    [AccessTokenStatus.ACCESS_TOKEN_STATUS_UNKNOWN]: 'gray.5',
  }

  const status = row.original.status ?? AccessTokenStatus.ACCESS_TOKEN_STATUS_UNKNOWN

  return (
    <Group gap={6}>
      <Box w={12} h={12} bdrs="50%" bg={statusColor[status]} />
      <Text size="sm">{t(`profile.status.${status}`)}</Text>
    </Group>
  )
}

const ActionCell = ({
  row, table,
}: DataTableRowActionsProps<AccessToken>) => {
  const { t } = useTranslation()

  const handleDeleteOpen = (
    table.options.meta as { handleDeleteOpen?: (token: AccessToken) => void } | undefined
  )?.handleDeleteOpen

  return (
    <Anchor
      component="button"
      size="sm"
      onClick={() => handleDeleteOpen?.(row.original)}
    >
      {t('profile.deleteToken')}
    </Anchor>
  )
}

export function AccessTokenTable({ tokens }: AccessTokenTableProps) {
  const { t } = useTranslation()
  const [deleteOpened, {
    open: openDelete, close: closeDelete,
  }] = useDisclosure(false)
  const [deletingToken, setDeletingToken] = useState<AccessToken | null>(null)

  const {
    mutate: deleteToken, isPending: isDeleting,
  } = useMutation({
    mutationFn: () => CurrentUser.DeleteAccessToken({ id: deletingToken?.id }),
    meta: {
      successMessage: t('profile.tokenDeleted'),
      invalidates: [profileKeys.accessTokens],
    },
    onSuccess: () => {
      closeDelete()
      setDeletingToken(null)
    },
  })

  const handleDeleteOpen = (token: AccessToken) => {
    setDeletingToken(token)
    openDelete()
  }

  const handleDeleteClose = () => {
    closeDelete()
    setDeletingToken(null)
  }

  const columns: MRT_ColumnDef<AccessToken>[] = [
    {
      accessorKey: 'name',
      header: t('profile.tokenName'),
    },
    {
      accessorKey: 'status',
      header: t('profile.tokenStatus'),
      Cell: StatusCell,
    },
    {
      accessorKey: 'expiredAt',
      header: t('profile.tokenExpiredAt'),
      Cell: ({ row }) => formatExpiredAt(row.original.expiredAt, t),
    },
    {
      accessorKey: 'createdAt',
      header: t('profile.tokenCreatedAt'),
      Cell: ({ row }) => formatDateTime(row.original.createdAt),
    },
  ]

  return (
    <>
      <DataTable
        columns={columns}
        data={tokens}
        enableRowActions
        renderRowActions={ActionCell}
        tableOptions={{
          meta: { handleDeleteOpen },
        }}
      />

      <ModalWrapper
        type="danger"
        title={t('profile.deleteToken')}
        opened={deleteOpened}
        onClose={handleDeleteClose}
        onConfirm={() => deleteToken()}
        confirmLoading={isDeleting}
      >
        <Text size="sm">
          {t('profile.deleteTokenConfirm', { name: deletingToken?.name ?? '' })}
        </Text>
      </ModalWrapper>
    </>
  )
}

import { Button } from '@mantine/core'
import { useDisclosure } from '@mantine/hooks'
import { useQueryClient, useSuspenseQuery } from '@tanstack/react-query'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'

import { CreateSshKeyModal } from '@/features/profile/components/CreateSshKeyModal'
import { DeleteSshKeyModal } from '@/features/profile/components/DeleteSshKeyModal'
import { SshKeysTable } from '@/features/profile/components/SshKeysTable'
import { profileKeys, sshKeysQueryOptions } from '@/features/profile/profile.query'

import type { SSHKey } from '@matrixhub/api-ts/v1alpha1/current_user.pb'

export function SshKeysPage() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const {
    data,
    isFetching,
  } = useSuspenseQuery(sshKeysQueryOptions())

  const [createOpened, createHandlers] = useDisclosure(false)
  const [deleteOpened, deleteHandlers] = useDisclosure(false)
  const [deleteTarget, setDeleteTarget] = useState<SSHKey | null>(null)

  const handleDelete = (key: SSHKey) => {
    setDeleteTarget(key)
    deleteHandlers.open()
  }

  const handleRefresh = () => {
    void queryClient.invalidateQueries({ queryKey: profileKeys.sshKeys })
  }

  const handleDeleteClose = () => {
    deleteHandlers.close()
    setDeleteTarget(null)
  }

  return (
    <>
      <SshKeysTable
        data={data.items ?? []}
        onDelete={handleDelete}
        loading={false}
        fetching={isFetching}
        onRefresh={handleRefresh}
        toolbarExtra={(
          <Button
            h={32}
            px="md"
            radius={6}
            onClick={createHandlers.open}
          >
            {t('profile.sshKey.create.button')}
          </Button>
        )}
      />

      <CreateSshKeyModal
        opened={createOpened}
        onClose={createHandlers.close}
      />

      <DeleteSshKeyModal
        sshKey={deleteTarget}
        opened={deleteOpened}
        onClose={handleDeleteClose}
      />
    </>
  )
}

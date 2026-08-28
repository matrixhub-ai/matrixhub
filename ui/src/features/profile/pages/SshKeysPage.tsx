import {
  Alert,
  Anchor,
  Button,
  Stack,
} from '@mantine/core'
import { useDisclosure } from '@mantine/hooks'
import {
  IconExternalLink,
  IconInfoCircle,
} from '@tabler/icons-react'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'

import { CreateSshKeyModal } from '@/features/profile/components/CreateSshKeyModal'
import { DeleteSshKeyModal } from '@/features/profile/components/DeleteSshKeyModal'
import { SshKeysTable } from '@/features/profile/components/SshKeysTable'
import { profileKeys, sshKeysQueryOptions } from '@/features/profile/profile.query'

import type { SSHKey } from '@matrixhub/api-ts/v1alpha1/current_user.pb'

const SSH_KEY_DOC_URL = '/docs/operations/profile/ssh-key/'

export function SshKeysPage() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const {
    data,
    isFetching,
  } = useQuery(sshKeysQueryOptions())

  const [createOpened, createHandlers] = useDisclosure(false)
  const [deleteOpened, deleteHandlers] = useDisclosure(false)
  const [hintVisible, setHintVisible] = useState(true)
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
    <Stack gap="sm">
      <Alert
        hidden={!hintVisible}
        icon={<IconInfoCircle size={20} />}
        variant="light"
        color="cyan"
        withCloseButton
        onClose={() => setHintVisible(false)}
        styles={{ icon: { marginRight: 6 } }}
      >
        {t('profile.sshKey.hint')}
        {' '}
        <Anchor
          href={t('common.docs', { doc: SSH_KEY_DOC_URL })}
          target="_blank"
          rel="noopener noreferrer"
          inherit
        >
          {t('profile.sshKey.generateHint')}
          {' '}
          <IconExternalLink size={14} />
        </Anchor>
      </Alert>

      <SshKeysTable
        data={data?.items ?? []}
        onDelete={handleDelete}
        loading={false}
        fetching={isFetching}
        onRefresh={handleRefresh}
        emptyTitle={t('profile.sshKey.emptyTitle')}
        toolbarExtra={(
          <Button
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
    </Stack>
  )
}

import { Button } from '@mantine/core'
import { useDisclosure } from '@mantine/hooks'
import { IconHomePlus } from '@tabler/icons-react'
import { useTranslation } from 'react-i18next'

import { RegistryFormModal } from './RegistryFormModal'

export function CreateRegistryAction() {
  const { t } = useTranslation()
  const [opened, {
    open,
    close,
  }] = useDisclosure(false)

  return (
    <>
      <Button
        onClick={open}
        leftSection={<IconHomePlus size={16} />}
      >
        {t('routes.admin.registries.toolbar.create')}
      </Button>

      {opened && (
        <RegistryFormModal
          mode="create"
          opened
          onClose={close}
        />
      )}
    </>
  )
}

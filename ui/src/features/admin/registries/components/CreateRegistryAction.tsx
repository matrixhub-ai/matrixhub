import { Button } from '@mantine/core'
import { useDisclosure } from '@mantine/hooks'
import { IconHomePlus } from '@tabler/icons-react'
import { getRouteApi } from '@tanstack/react-router'
import { useEffect, useEffectEvent } from 'react'
import { useTranslation } from 'react-i18next'

import { RegistryFormModal } from './RegistryFormModal'

const registriesRouteApi = getRouteApi('/(auth)/admin/registries')

export function CreateRegistryAction() {
  const { t } = useTranslation()
  const navigate = registriesRouteApi.useNavigate()
  const { create } = registriesRouteApi.useSearch()
  const [opened, {
    open,
    close,
  }] = useDisclosure(false)

  // `?create=true` (e.g. from project creation) opens the dialog on arrival.
  // The param is dropped right away so a later reload does not reopen it.
  const openFromSearch = useEffectEvent(() => {
    if (!create) {
      return
    }

    open()
    void navigate({
      search: prev => ({
        ...prev,
        create: undefined,
      }),
      replace: true,
    })
  })

  useEffect(() => {
    openFromSearch()
  }, [create])

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

import { Button } from '@mantine/core'
import {
  IconArrowRight,
  IconCloudDownload,
} from '@tabler/icons-react'
import { useTranslation } from 'react-i18next'

import { resolveServerPullFooter } from '../server-pull.utils'

import type { ServerPull } from '../hooks/useServerPull'

interface ServerPullActionsProps {
  pull: ServerPull
  onClose: () => void
  /** Offers a link to the pulled model; leave it out where the model is already shown. */
  onViewModel?: () => void
}

/** Footer buttons of the server pull: cancel or close, and start, retry or view the model. */
export function ServerPullActions({
  pull,
  onClose,
  onViewModel,
}: ServerPullActionsProps) {
  const { t } = useTranslation()
  const {
    primary,
    dismiss,
  } = resolveServerPullFooter(pull.view, onViewModel !== undefined)

  return (
    <>
      <Button variant="default" onClick={onClose}>
        {dismiss === 'cancel' ? t('common.cancel') : t('model.serverPull.actions.close')}
      </Button>
      {primary?.action === 'viewModel' && (
        <Button rightSection={<IconArrowRight size={16} />} onClick={onViewModel}>
          {t('model.serverPull.actions.viewModel')}
        </Button>
      )}
      {(primary?.action === 'start' || primary?.action === 'retry') && (
        <Button
          leftSection={<IconCloudDownload size={16} />}
          loading={pull.starting}
          disabled={primary.disabled}
          onClick={pull.start}
        >
          {t(`model.serverPull.actions.${primary.action}`)}
        </Button>
      )}
    </>
  )
}

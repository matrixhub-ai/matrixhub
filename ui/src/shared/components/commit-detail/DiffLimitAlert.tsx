import {
  Alert,
  Anchor,
  Text,
} from '@mantine/core'
import { type MouseEvent } from 'react'
import {
  Trans,
  useTranslation,
} from 'react-i18next'

interface DiffLimitAlertProps {
  maxFiles: number
  onOpenRawDiff: (event: MouseEvent<HTMLAnchorElement>) => void
}

export function DiffLimitAlert({
  maxFiles,
  onOpenRawDiff,
}: DiffLimitAlertProps) {
  const { t } = useTranslation()

  return (
    <Alert variant="light" color="yellow">
      <Text size="sm" component="span">
        <Trans
          t={t}
          i18nKey="shared.commitDetail.diffLimitNoticeWithLink"
          values={{ count: maxFiles }}
          components={[
            <Anchor
              key="raw-diff-link"
              size="sm"
              href="#"
              onClick={onOpenRawDiff}
            />,
          ]}
        />
      </Text>
    </Alert>
  )
}

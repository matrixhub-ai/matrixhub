import {
  Box,
  Button,
  Code,
  Group,
  Stack,
  Text,
} from '@mantine/core'
import { useTranslation } from 'react-i18next'

import { CopyValueButton } from '@/shared/components/CopyValueButton'
import { ModalWrapper } from '@/shared/components/ModalWrapper'

import classes from './ModelCommandDialog.module.css'

export type ModelCommandType = 'upload' | 'download' | 'use'

interface ModelCommandDialogProps {
  opened: boolean
  type: ModelCommandType
  modelPath: string
  onClose: () => void
}

interface CommandCodeBlockProps {
  command: string
}

const HF_ENDPOINT_PLACEHOLDER = 'https://example.matrixhub.io'

function buildModelCommand(type: ModelCommandType, modelPath: string) {
  const commandByType = {
    upload: `hf upload ${modelPath} . .`,
    download: `hf download ${modelPath}`,
    use: `vllm serve ${modelPath}`,
  }

  return `export HF_ENDPOINT=${HF_ENDPOINT_PLACEHOLDER}\n${commandByType[type]}`
}

function CommandCodeBlock({ command }: CommandCodeBlockProps) {
  return (
    <Box className={classes.codeBlock}>
      <CopyValueButton
        value={command}
        timeout={1500}
        iconClassName={classes.copyButton}
      >
      </CopyValueButton>

      <Code block className={classes.code}>
        {command}
      </Code>
    </Box>
  )
}

export function ModelCommandDialog({
  opened,
  type,
  modelPath,
  onClose,
}: ModelCommandDialogProps) {
  const { t } = useTranslation()
  const command = buildModelCommand(type, modelPath)

  return (
    <ModalWrapper
      title={t(`model.detail.commandDialog.${type}.title`)}
      opened={opened}
      onClose={onClose}
      size="lg"
      footer={(
        <Group justify="flex-end">
          <Button onClick={onClose}>
            {t('common.confirm')}
          </Button>
        </Group>
      )}
    >
      <Stack gap="md">
        <Text size="sm" c="dimmed">
          {t(`model.detail.commandDialog.${type}.description`)}
        </Text>

        <CommandCodeBlock command={command} />
      </Stack>
    </ModalWrapper>
  )
}

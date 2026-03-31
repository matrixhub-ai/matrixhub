import {
  Alert,
  Button,
  Center, Group, Stack, Text,
} from '@mantine/core'
import { IconAlertCircle, IconRefresh } from '@tabler/icons-react'
import { useQueryClient } from '@tanstack/react-query'
import { useRouter } from '@tanstack/react-router'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'

import { getErrorMessage } from '@/queryClient'

interface RouterErrorComponentProps {
  error: unknown
}

export function RouterErrorComponent({ error }: RouterErrorComponentProps) {
  const { t } = useTranslation()
  const router = useRouter()
  const queryClient = useQueryClient()
  const [isReloading, setIsReloading] = useState(false)

  const handleReload = async () => {
    try {
      setIsReloading(true)
      // 1. Reset all queries in the query client
      await queryClient.resetQueries()

      // 2. Invalidate the current route to re-run the loader
      await router.invalidate()
    } catch (error) {
      console.error('Failed to reload router state:', error)
    } finally {
      setIsReloading(false)
    }
  }

  return (
    <Center h="100%" p="xl" style={{ flexGrow: 1 }}>
      <Stack align="center" gap="lg" maw={500}>
        <Alert
          variant="light"
          color="red"
          title={t('shared.components.RouterErrorComponent.somethingWentWrong')}
          icon={<IconAlertCircle size={20} />}
          w="100%"
        >
          <Text size="sm" style={{ wordBreak: 'break-word' }}>
            {getErrorMessage(error)}
          </Text>
        </Alert>
        <Group>
          <Button
            variant="outline"
            leftSection={<IconRefresh size={16} />}
            loading={isReloading}
            onClick={() => void handleReload()}
          >
            {t('shared.components.RouterErrorComponent.reload')}
          </Button>
          <Button
            variant="filled"
            disabled={isReloading}
            onClick={() => {
              router.navigate({ to: '/' })
            }}
          >
            {t('shared.components.RouterErrorComponent.backToHome')}
          </Button>
        </Group>
      </Stack>
    </Center>
  )
}

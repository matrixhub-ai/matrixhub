import {
  ActionIcon,
  Alert,
  Button,
  Flex,
  Group,
  rem,
  Stack,
  Text,
  TextInput,
} from '@mantine/core'
import { useDisclosure } from '@mantine/hooks'
import { CurrentUser, type CreateAccessTokenRequest } from '@matrixhub/api-ts/v1alpha1/current_user.pb'
import {
  IconInfoCircle,
  IconKey,
  IconRefresh,
} from '@tabler/icons-react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import dayjs from 'dayjs'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import z from 'zod'

import { AccessTokenTable } from '@/features/profile/components/AccessTokenTable'
import { ExpireAtField } from '@/features/profile/components/ExpireAtField'
import { profileKeys, useAccessTokens } from '@/features/profile/profile.query'
import { expireAtSchema } from '@/features/profile/profile.schema.ts'
import { CopyValueButton } from '@/shared/components/CopyValueButton'
import { ModalWrapper } from '@/shared/components/ModalWrapper'
import { useForm } from '@/shared/hooks/useForm'

export function AccessTokenPage() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()

  const {
    data, isFetching,
  } = useAccessTokens()
  const tokens = data?.items ?? []

  const [hintVisible, setHintVisible] = useState(true)
  const [createOpened, {
    open: openCreate, close: closeCreate,
  }] = useDisclosure(false)

  const [copyOpened, {
    open: openCopy, close: closeCopy,
  }] = useDisclosure(false)

  const handleRefresh = () => {
    queryClient.invalidateQueries({ queryKey: profileKeys.accessTokens })
  }

  const nameSchema = z.string().min(1, { error: t('common.validation.fieldRequired', { field: t('profile.tokenName') }) })

  const [newToken, setNewToken] = useState<string>('')

  const {
    mutate: createToken, isPending: isCreating,
  } = useMutation({
    mutationFn: (value: CreateAccessTokenRequest) => CurrentUser.CreateAccessToken(value),
    meta: {
      successMessage: t('profile.tokenCreated'),
      invalidates: [profileKeys.accessTokens],
    },
    onSuccess: (res) => {
      setNewToken(res.token ?? '')
      handleCreateClose()

      if (res.token) {
        openCopy()
      }
    },
  })

  const form = useForm({
    defaultValues: {
      name: '',
      expireAt: '',
    },
    onSubmit: ({ value }) => {
      createToken({
        ...value,
        expireAt: value.expireAt ? String(dayjs(value.expireAt).unix()) : '',
      })
    },
  })

  const handleCreateClose = () => {
    closeCreate()
    form.reset()
  }

  const handleCopyClose = () => {
    closeCopy()
    setNewToken('')
  }

  return (
    <Stack gap="sm">
      {hintVisible && (
        <Alert
          icon={<IconInfoCircle size={20} />}
          variant="light"
          color="cyan"
          withCloseButton
          onClose={() => setHintVisible(false)}
          styles={{ icon: { marginRight: 6 } }}
        >
          <Text size="sm">{t('profile.tokenHint')}</Text>
        </Alert>
      )}

      <Group justify="flex-end" gap={16}>
        <ActionIcon
          variant="transparent"
          loading={isFetching}
          onClick={handleRefresh}
          aria-label="refresh"
          c="gray.6"
          size={24}
        >
          <IconRefresh />
        </ActionIcon>
        <Button
          leftSection={<IconKey size={16} />}
          onClick={openCreate}
          size="xs"
        >
          {t('profile.createToken')}
        </Button>
      </Group>

      <AccessTokenTable tokens={tokens} />

      <ModalWrapper
        title={t('profile.createToken')}
        opened={createOpened}
        onClose={handleCreateClose}
        onConfirm={form.handleSubmit}
        confirmLoading={isCreating}
        size="sm"
      >
        <form.Field
          name="name"
          validators={{ onChange: nameSchema }}
        >
          {({
            state, handleChange,
          }) => (
            <TextInput
              label={t('profile.tokenName')}
              required
              value={state.value}
              onChange={e => handleChange(e.currentTarget.value)}
              error={state.meta.errors[0]?.message}
            />
          )}
        </form.Field>
        <form.Field name="expireAt" validators={{ onChange: expireAtSchema }}>
          {({
            state, handleChange,
          }) => (
            <ExpireAtField
              value={state.value}
              onChange={handleChange}
              error={state.meta.errors[0]?.message}
            />
          )}
        </form.Field>
      </ModalWrapper>

      <ModalWrapper
        title={t('profile.copyTitle')}
        opened={copyOpened}
        onClose={handleCopyClose}
        onConfirm={handleCopyClose}
        footer={(
          <Group justify="flex-end">
            <Button onClick={handleCopyClose}>
              {t('common.confirm')}
            </Button>
          </Group>
        )}
        size="md"
      >
        <Stack gap="md">
          <Alert
            variant="light"
            color="cyan"
            bdrs="sm"
            p={12}
            lh={rem(20)}
            styles={{ icon: { marginRight: 8 } }}
            icon={<IconInfoCircle size={20} />}
          >
            {t('profile.copyDescription')}
          </Alert>

          <Flex bg="gray.0" c="gray.9" justify="space-between" px={11} py={8} bdrs="md" lh={rem(20)}>
            <Text size="sm">{newToken}</Text>
            <CopyValueButton color="dark" iconSize={14} value={newToken} />
          </Flex>
        </Stack>
      </ModalWrapper>
    </Stack>
  )
}

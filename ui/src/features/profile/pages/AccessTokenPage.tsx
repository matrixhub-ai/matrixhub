import {
  ActionIcon,
  Alert,
  Button,
  Flex,
  Group,
  Radio,
  rem,
  Stack,
  Text,
  TextInput,
} from '@mantine/core'
import { DatePickerInput } from '@mantine/dates'
import { useDisclosure } from '@mantine/hooks'
import { CurrentUser, type CreateAccessTokenRequest } from '@matrixhub/api-ts/v1alpha1/current_user.pb'
import {
  IconInfoCircle,
  IconKey,
  IconRefresh,
} from '@tabler/icons-react'
import { useForm } from '@tanstack/react-form'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import dayjs from 'dayjs'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import z from 'zod'

import { AccessTokenTable } from '@/features/profile/components/AccessTokenTable'
import { profileKeys, useAccessTokens } from '@/features/profile/profile.query'
import { CopyValueButton } from '@/shared/components/CopyValueButton'
import { ModalWrapper } from '@/shared/components/ModalWrapper'

const validityRadios = [
  {
    value: 'never',
    label: 'profile.expireNever',
  },
  {
    value: 'custom',
    label: 'profile.expireCustom',
  },
] as const

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

  const [validityValue, setValidityValue] = useState<'never' | 'custom'>('never')

  const handleRefresh = () => {
    queryClient.invalidateQueries({ queryKey: profileKeys.accessTokens })
  }

  const nameSchema = z.string().min(1, { error: t('common.validation.fieldRequired', { field: t('profile.tokenName') }) })

  const expireAtSchema = z.string().min(1, { error: t('common.validation.fieldRequired', { field: t('profile.expireTime') }) })
    .refine(value => dayjs(value).startOf('day').isAfter(dayjs().startOf('day')), { error: t('profile.expireTimeError') })

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
      expireAt: dayjs().add(1, 'day').format('YYYY-MM-DD'),
    },
    onSubmit: ({ value }) => {
      createToken({
        ...value,
        expireAt: validityValue === 'custom' ? String(dayjs(value.expireAt).unix()) : '',
      })
    },
  })

  const handleValidityChange = (v: string) => {
    setValidityValue(v as 'never' | 'custom')

    if (v === 'never') {
      form.resetField('expireAt')
    }
  }

  const handleCreateClose = () => {
    closeCreate()
    form.reset()
    setValidityValue('never')
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
        <Radio.Group
          label={t('profile.validity')}
          name="validity"
          value={validityValue}
          onChange={handleValidityChange}
        >
          <Group mt="xs">
            {validityRadios.map(radio => (
              <Radio
                key={radio.value}
                value={radio.value}
                label={t(radio.label)}
              />
            ))}
          </Group>
        </Radio.Group>
        {validityValue === 'custom' && (
          <form.Field
            name="expireAt"
            validators={{ onChange: expireAtSchema }}
          >
            {({
              state, handleChange,
            }) => (
              <DatePickerInput
                label={t('profile.expireTime')}
                valueFormat="YYYY-MM-DD"
                value={state.value}
                required
                highlightToday
                onChange={e => handleChange(e ?? '')}
                error={state.meta.errors[0]?.message}
              />
            )}
          </form.Field>
        )}
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

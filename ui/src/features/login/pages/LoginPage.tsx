import {
  Button, Checkbox, Flex, Group, PasswordInput, ScrollArea, Stack, Text, TextInput, rem,
} from '@mantine/core'
import { Login, type LoginRequest } from '@matrixhub/api-ts/v1alpha1/login.pb'
import { useMutation } from '@tanstack/react-query'
import { useNavigate } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'

import LogoIcon from '@/assets/svgs/logo.svg?react'
import { LanguageSwitcher } from '@/shared/components/LanguageSwitcher'
import { useForm } from '@/shared/hooks/useForm'

export function LoginPage() {
  const { t } = useTranslation()
  const navigate = useNavigate()

  const {
    mutate: handleLogin, isPending: isLoggingIn,
  } = useMutation({
    mutationFn: (values: LoginRequest) => Login.Login(values),
    onSuccess: () => {
      navigate({ to: '/' })
    },
  })

  const form = useForm({
    defaultValues: {
      username: '',
      password: '',
      rememberMe: true,
    },
    onSubmit: (values) => {
      handleLogin(values.value)
    },
  })

  return (
    <ScrollArea h="100vh" bg="gray.1" pt="24">
      <Flex justify="flex-end" px="36">
        <LanguageSwitcher />
      </Flex>

      <form
        style={{ width: '100%' }}
        onSubmit={(e) => {
          e.preventDefault()
          form.handleSubmit()
        }}
      >
        <Stack p="md" bg="white" w="388" align="center" gap="xl" bdrs="md" ml="auto" mr="12vw" mt="100" mb="24">
          <Group
            gap={8}
            wrap="nowrap"
          >
            <LogoIcon fontSize={rem(36)} />
            <Text
              fw={600}
              size="xl"
              c="var(--mantine-color-black)"
            >
              MatrixHub
            </Text>
          </Group>

          <Text fw="600" lh="2" fz="20">{t('login.title')}</Text>

          <form.Field
            name="username"
          >
            {field => (
              <TextInput
                label={t('login.username')}
                required
                w="100%"
                size="md"
                name={field.name}
                value={field.state.value}
                onChange={e => field.handleChange(e.target.value)}
              />
            )}
          </form.Field>

          <form.Field
            name="password"
          >
            {field => (
              <PasswordInput
                label={t('login.password')}
                required
                w="100%"
                size="md"
                name={field.name}
                value={field.state.value}
                onChange={e => field.handleChange(e.target.value)}
              />
            )}
          </form.Field>

          <form.Field name="rememberMe">
            {field => (
              <Checkbox
                name={field.name}
                checked={field.state.value}
                onChange={e => field.handleChange(e.target.checked)}
                label={t('login.rememberMe')}
                style={{ alignSelf: 'flex-start' }}
              />
            )}
          </form.Field>

          <Button type="submit" style={{ alignSelf: 'flex-end' }} loading={isLoggingIn}>{t('login.submit')}</Button>
        </Stack>
      </form>
    </ScrollArea>
  )
}

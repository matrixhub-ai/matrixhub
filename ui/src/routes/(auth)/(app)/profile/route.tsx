import {
  Box, Group, Stack, Tabs, Text,
} from '@mantine/core'
import {
  createFileRoute, Link, Outlet, useMatchRoute,
} from '@tanstack/react-router'
import { use } from 'react'
import { useTranslation } from 'react-i18next'

import DefaultAvatarIcon from '@/assets/svgs/default-avatar.svg?react'
import UserIcon from '@/assets/svgs/user.svg?react'
import { CurrentUserContext } from '@/context/current-user-context.tsx'
import { Route as AccessTokenRoute } from '@/routes/(auth)/(app)/profile/access-token'
import { Route as LanguageRoute } from '@/routes/(auth)/(app)/profile/language'
import { Route as SecurityRoute } from '@/routes/(auth)/(app)/profile/security'

export const Route = createFileRoute('/(auth)/(app)/profile')({
  component: Profile,
})

function Profile() {
  const { t } = useTranslation()
  const currentUser = use(CurrentUserContext)

  const profileTabs = [
    {
      label: t('profile.securitySetting'),
      value: 'security',
      to: SecurityRoute.to,
    },
    {
      label: t('profile.accessToken'),
      value: 'access-token',
      to: AccessTokenRoute.to,
    },
    {
      label: t('profile.languageSetting'),
      value: 'language',
      to: LanguageRoute.to,
    },
  ]

  const matchRoute = useMatchRoute()
  const activeTab = profileTabs.find(tab => matchRoute({ to: tab.to }))?.value || profileTabs[0].value

  return (
    <Stack gap="lg">
      <Group
        gap="xs"
        px={28}
      >
        <UserIcon
          width={32}
          height={32}
        />
        <Text
          size="lg"
          fw={600}
        >
          { t('profile.personalCenter') }
        </Text>
      </Group>

      <Group
        gap="md"
        h={80}
        px={20}
        bdrs={16}
        bg="#F8F9FA"
      >
        <DefaultAvatarIcon width={48} height={48} />

        <Text
          size="sm"
          fw={600}
        >
          {currentUser?.username ?? ''}
        </Text>
      </Group>

      <Tabs
        value={activeTab}
      >
        <Tabs.List>
          {profileTabs.map((tab) => {
            return (
              // TODO: tab style should keep same in different page
              <Tabs.Tab
                key={tab.value}
                value={tab.value}
                renderRoot={props => (
                  <Link
                    to={tab.to}
                    {...props}
                    style={{
                      ...(props.style ?? {}),
                      marginRight: '16px',
                    }}
                  />
                )}
              >
                {tab.label}
              </Tabs.Tab>
            )
          })}
        </Tabs.List>

        <Box pt={20}>
          <Outlet />
        </Box>
      </Tabs>
    </Stack>
  )
}

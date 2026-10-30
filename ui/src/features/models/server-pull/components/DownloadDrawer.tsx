import {
  Box,
  Drawer,
  Group,
  Stack,
  Text,
} from '@mantine/core'

import classes from './DownloadDrawer.module.css'

import type { ReactNode } from 'react'

interface DownloadDrawerProps {
  opened: boolean
  title: ReactNode
  /** Dimmed line under the title, such as the target project or model. */
  subtitle?: ReactNode
  onClose: () => void
  /**
   * A `DownloadDrawerBody` and a `DownloadDrawerFooter`. They render inside the drawer, so
   * their state resets each time the drawer opens.
   */
  children: ReactNode
}

/** Right-side drawer of the download flows: a form-like body and a footer with a hint and actions. */
export function DownloadDrawer({
  opened,
  title,
  subtitle,
  onClose,
  children,
}: DownloadDrawerProps) {
  // The drawer renders its title in a heading, so it may only hold phrasing content.
  const heading = (
    <Stack component="span" gap={4}>
      <Text component="span" fw={600} fz="lg">{title}</Text>
      {subtitle && <Text component="span" size="xs" c="dimmed">{subtitle}</Text>}
    </Stack>
  )

  return (
    <Drawer
      opened={opened}
      onClose={onClose}
      position="right"
      size="lg"
      title={heading}
      classNames={{
        content: classes.drawerContent,
        header: classes.header,
        body: classes.body,
      }}
    >
      {children}
    </Drawer>
  )
}

export function DownloadDrawerBody({ children }: { children: ReactNode }) {
  return (
    <Box className={classes.content}>
      <Stack gap="lg">{children}</Stack>
    </Box>
  )
}

export function DownloadDrawerFooter({
  hint,
  children,
}: {
  /** Dimmed note at the left, such as what happens after the drawer closes. */
  hint?: ReactNode
  /** Buttons, from the least to the most prominent. */
  children: ReactNode
}) {
  return (
    <Group className={classes.footer} justify="space-between" wrap="nowrap" gap="md">
      <Text size="xs" c="dimmed">{hint}</Text>
      <Group gap="sm" wrap="nowrap" flex="none">{children}</Group>
    </Group>
  )
}

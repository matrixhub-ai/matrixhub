import {
  Alert,
  Badge,
  Box,
  Button,
  Drawer,
  Group,
  Stack,
  Text,
} from '@mantine/core'
import { IconInfoCircle } from '@tabler/icons-react'
import { useTranslation } from 'react-i18next'

import { ShikiCodeBlock } from '@/shared/components/ShikiCodeBlock'

import classes from './GuideDrawer.module.css'

import type { ReactNode } from 'react'

interface GuideDrawerProps {
  opened: boolean
  title: ReactNode
  /** Short explanation shown in an info alert above the steps. */
  intro: ReactNode
  /** Rendered at the left of the footer, opposite the confirm button. */
  footerExtra?: ReactNode
  onClose: () => void
  children: ReactNode
}

/**
 * Right-side drawer for step-by-step command guides (use / upload / download).
 * The body scrolls; the footer stays pinned with a confirm button.
 */
export function GuideDrawer({
  opened,
  title,
  intro,
  footerExtra,
  onClose,
  children,
}: GuideDrawerProps) {
  const { t } = useTranslation()

  return (
    <Drawer
      opened={opened}
      onClose={onClose}
      position="right"
      size="lg"
      title={<Text fw={600} fz="md">{title}</Text>}
      classNames={{ body: classes.body }}
    >
      <Box className={classes.content}>
        <Stack gap="md">
          <Alert
            variant="light"
            color="cyan"
            icon={<IconInfoCircle size={16} />}
            py="xs"
          >
            {intro}
          </Alert>
          {children}
        </Stack>
      </Box>

      <Group justify={footerExtra ? 'space-between' : 'flex-end'} className={classes.footer}>
        {footerExtra}
        <Button onClick={onClose}>{t('common.confirm')}</Button>
      </Group>
    </Drawer>
  )
}

interface GuideStepProps {
  index: number
  title: ReactNode
  /** Rendered at the right of the step header (e.g. a selector). */
  extra?: ReactNode
  /** Dimmed helper line under the step content. */
  hint?: ReactNode
  children: ReactNode
}

export function GuideStep({
  index, title, extra, hint, children,
}: GuideStepProps) {
  return (
    <Box className={classes.step}>
      <Group className={classes.stepHeader} justify="space-between" wrap="nowrap">
        <Group gap="xs" wrap="nowrap">
          <Badge circle size="md" variant="filled" className={classes.stepIndex}>
            {index}
          </Badge>
          <Text component="div" fw={600} size="sm">{title}</Text>
        </Group>
        {extra}
      </Group>
      <Stack gap="xs" className={classes.stepBody}>
        {children}
        {hint && <Text size="xs" c="dimmed">{hint}</Text>}
      </Stack>
    </Box>
  )
}

export interface GuideSnippet {
  code: string
  lang: 'bash' | 'python'
}

export function GuideSnippetBlock({ snippet }: { snippet: GuideSnippet }) {
  return <ShikiCodeBlock code={snippet.code} lang={snippet.lang} />
}

/** Bordered note card, used for closing remarks such as where files end up. */
export function GuideNote({ children }: { children: ReactNode }) {
  return (
    <Box className={classes.note}>
      <Text size="sm">{children}</Text>
    </Box>
  )
}

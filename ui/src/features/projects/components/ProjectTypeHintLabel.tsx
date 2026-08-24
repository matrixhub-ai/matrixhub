import {
  Anchor, Group, HoverCard, List, Text,
} from '@mantine/core'
import { IconExternalLink } from '@tabler/icons-react'
import { Trans, useTranslation } from 'react-i18next'

import IconQuestion from '@/assets/svgs/question.svg?react'

import type { ReactNode } from 'react'

// TODO: replace with the permission guide path once the doc is published.
// Tracked by https://github.com/matrixhub-ai/matrixhub/issues/872
const PROJECT_PERMISSION_DOC_URL = '/docs/operations/project-management/members/'

const DROPDOWN_WIDTH = 360

/**
 * Anchor that appends an external-link icon after the translated label.
 *
 * The icon cannot be an empty placeholder tag in the translation string:
 * Trans clones the mapped element with the tag's own children, so a
 * `<1></1>` placeholder renders as nothing.
 */
function PermissionDocLink({ children }: { children?: ReactNode }) {
  const { t } = useTranslation()

  return (
    <Anchor
      href={t('common.docs', { doc: PROJECT_PERMISSION_DOC_URL })}
      target="_blank"
      rel="noopener noreferrer"
      inherit
      c="blue.4"
    >
      {children}
      {' '}
      <IconExternalLink size={14} style={{ verticalAlign: '-0.2em' }} />
    </Anchor>
  )
}

interface ProjectTypeHintLabelProps {
  label: ReactNode
}

/**
 * Project visibility label with an interactive hint.
 *
 * Uses HoverCard instead of the shared FieldHintLabel tooltip because the hint
 * contains a link, and Mantine tooltips are `pointer-events: none`.
 */
export function ProjectTypeHintLabel({ label }: ProjectTypeHintLabelProps) {
  const { t } = useTranslation()

  return (
    <Group
      component="span"
      gap={4}
      align="center"
      wrap="nowrap"
      style={{
        display: 'inline-flex',
        alignItems: 'center',
      }}
    >
      <Text component="span" inherit>
        {label}
      </Text>
      <HoverCard
        width={DROPDOWN_WIDTH}
        shadow="md"
        withArrow
        openDelay={100}
        closeDelay={150}
        styles={{
          dropdown: {
            backgroundColor: 'var(--mantine-color-gray-9)',
            borderColor: 'var(--mantine-color-gray-9)',
            color: 'var(--mantine-color-white)',
          },
          arrow: {
            backgroundColor: 'var(--mantine-color-gray-9)',
            borderColor: 'var(--mantine-color-gray-9)',
          },
        }}
      >
        <HoverCard.Target>
          <IconQuestion
            width={18}
            height={18}
            style={{
              cursor: 'help',
              flex: 'none',
            }}
          />
        </HoverCard.Target>
        <HoverCard.Dropdown>
          <List size="xs" spacing={4}>
            <List.Item>{t('projects.typeHint.public')}</List.Item>
            <List.Item>{t('projects.typeHint.private')}</List.Item>
            <List.Item>
              <Trans
                i18nKey="projects.typeHint.permission"
                components={[<PermissionDocLink key="doc" />]}
              />
            </List.Item>
          </List>
        </HoverCard.Dropdown>
      </HoverCard>
    </Group>
  )
}

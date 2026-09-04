import {
  Anchor, Group, List, Popover, Text,
} from '@mantine/core'
import { IconExternalLink } from '@tabler/icons-react'
import { useState } from 'react'
import { Trans, useTranslation } from 'react-i18next'

import IconQuestion from '@/assets/svgs/question.svg?react'

import type { KeyboardEvent, ReactNode } from 'react'

// TODO: point this at the permission guide once that doc is published. It
// currently falls back to the project members doc, which covers role
// permissions but not the full ruleset.
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
      <IconExternalLink
        size={14}
        style={{
          verticalAlign: '-0.2em',
          marginInlineStart: '0.25em',
        }}
      />
    </Anchor>
  )
}

interface ProjectTypeHintLabelProps {
  label: ReactNode
}

/**
 * Project visibility label with an interactive hint.
 *
 * Uses Popover rather than the shared FieldHintLabel tooltip because the hint
 * contains a link: Mantine tooltips are `pointer-events: none`, so a link
 * inside one cannot be clicked. The open state is controlled so that the hint
 * responds to keyboard focus as well as hover.
 */
export function ProjectTypeHintLabel({ label }: ProjectTypeHintLabelProps) {
  const { t } = useTranslation()
  const [opened, setOpened] = useState(false)

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
      <Popover
        width={DROPDOWN_WIDTH}
        shadow="md"
        withArrow
        opened={opened}
        onChange={setOpened}
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
        <Popover.Target>
          <IconQuestion
            width={18}
            height={18}
            role="button"
            tabIndex={0}
            aria-label={t('common.moreInfo')}
            onMouseEnter={() => setOpened(true)}
            onMouseLeave={() => setOpened(false)}
            onFocus={() => setOpened(true)}
            onBlur={() => setOpened(false)}
            onKeyDown={(event: KeyboardEvent) => {
              if (event.key === 'Escape') {
                setOpened(false)
              }
            }}
            style={{
              cursor: 'help',
              flex: 'none',
            }}
          />
        </Popover.Target>
        <Popover.Dropdown
          onMouseEnter={() => setOpened(true)}
          onMouseLeave={() => setOpened(false)}
        >
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
        </Popover.Dropdown>
      </Popover>
    </Group>
  )
}

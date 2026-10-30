import {
  Combobox,
  type ComboboxItemGroup,
  InputBase,
  Select,
  Text,
  useCombobox,
} from '@mantine/core'
import { useTranslation } from 'react-i18next'

import classes from './RevisionFields.module.css'

import type { Revision } from '@matrixhub/api-ts/v1alpha1/model.pb'

interface RevisionFieldProps {
  /** Branch, tag or commit; empty means the default branch. */
  value: string
  onChange: (revision: string) => void
  onBlur: () => void
}

interface RevisionInputFieldProps extends RevisionFieldProps {
  /** Name of the upstream default branch, once a preview tells it. */
  defaultBranch?: string
}

/**
 * Revision of a model that may not be in MatrixHub yet, so its branches cannot be listed:
 * the default branch, or a branch, tag or commit typed by the user.
 */
export function RevisionInputField({
  value,
  onChange,
  onBlur,
  defaultBranch,
}: RevisionInputFieldProps) {
  const { t } = useTranslation()
  const combobox = useCombobox({
    onDropdownClose: () => combobox.resetSelectedOption(),
  })
  const typed = value.trim()
  const defaultLabel = defaultBranch
    ? t('model.serverPull.revision.defaultBranchOf', { branch: defaultBranch })
    : t('model.serverPull.revision.defaultBranch')

  return (
    <Combobox
      store={combobox}
      onOptionSubmit={(revision) => {
        onChange(revision)
        combobox.closeDropdown()
      }}
    >
      <Combobox.Target>
        <InputBase
          label={t('model.serverPull.revision.label')}
          withAsterisk
          description={t('model.serverPull.revision.description')}
          inputWrapperOrder={['label', 'input', 'description', 'error']}
          classNames={{ input: classes.input }}
          placeholder={defaultLabel}
          rightSection={<Combobox.Chevron />}
          rightSectionPointerEvents="none"
          value={value}
          onChange={(event) => {
            onChange(event.currentTarget.value)
            combobox.openDropdown()
            combobox.updateSelectedOptionIndex()
          }}
          onClick={() => combobox.openDropdown()}
          onFocus={() => combobox.openDropdown()}
          onBlur={() => {
            combobox.closeDropdown()
            onBlur()
          }}
        />
      </Combobox.Target>

      <Combobox.Dropdown>
        <Combobox.Options>
          <Combobox.Option value="" active={typed === ''}>
            {defaultLabel}
          </Combobox.Option>
          {typed && (
            <Combobox.Option value={typed} active>
              {t('model.serverPull.revision.useTyped', { revision: typed })}
            </Combobox.Option>
          )}
        </Combobox.Options>
        <Combobox.Footer>
          <Text size="xs" c="dimmed">{t('model.serverPull.revision.typeHint')}</Text>
        </Combobox.Footer>
      </Combobox.Dropdown>
    </Combobox>
  )
}

interface RevisionSelectFieldProps extends RevisionFieldProps {
  branches?: Revision[]
  tags?: Revision[]
  /** Listed first and preselected, also while the branches load. */
  defaultBranch: string
}

const BRANCH_PREFIX = 'branch:'
const TAG_PREFIX = 'tag:'
const NO_REVISIONS: Revision[] = []

/** Revision of a model in MatrixHub: one of its branches or tags. */
export function RevisionSelectField({
  value,
  onChange,
  onBlur,
  branches = NO_REVISIONS,
  tags = NO_REVISIONS,
  defaultBranch,
}: RevisionSelectFieldProps) {
  const { t } = useTranslation()
  const branchNames = uniqueNames([defaultBranch, ...branches.map(branch => branch.name)])
  const tagNames = uniqueNames(tags.map(tag => tag.name))
  // A branch and a tag can share a name, so option values carry the kind.
  const data: ComboboxItemGroup[] = [
    {
      group: t('shared.revisionSelect.group.branch'),
      items: branchNames.map(name => ({
        value: `${BRANCH_PREFIX}${name}`,
        label: name === defaultBranch
          ? t('model.serverPull.revision.defaultBranchOf', { branch: name })
          : name,
      })),
    },
  ]

  if (tagNames.length > 0) {
    data.push({
      group: t('shared.revisionSelect.group.tag'),
      items: tagNames.map(name => ({
        value: `${TAG_PREFIX}${name}`,
        label: name,
      })),
    })
  }

  let selected: string | null = null

  if (branchNames.includes(value)) {
    selected = `${BRANCH_PREFIX}${value}`
  } else if (tagNames.includes(value)) {
    selected = `${TAG_PREFIX}${value}`
  }

  return (
    <Select
      label={t('model.serverPull.revision.label')}
      withAsterisk
      description={t('model.serverPull.revision.description')}
      inputWrapperOrder={['label', 'input', 'description', 'error']}
      data={data}
      value={selected}
      searchable
      allowDeselect={false}
      onChange={(next) => {
        if (next) {
          onChange(next.slice(next.indexOf(':') + 1))
        }
      }}
      onBlur={onBlur}
    />
  )
}

function uniqueNames(names: (string | undefined)[]) {
  return [...new Set(names.filter((name): name is string => Boolean(name)))]
}

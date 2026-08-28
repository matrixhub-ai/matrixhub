import {
  CheckIcon,
  Combobox,
  Group,
  InputBase,
  type InputBaseProps,
  ScrollArea,
  Text,
  useCombobox,
} from '@mantine/core'
import { type ComponentProps, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { ProjectTypeBadge } from '@/shared/components/badges/ProjectTypeBadge'
import { FieldHintLabel } from '@/shared/components/FieldHintLabel.tsx'
import { filterByKeyword } from '@/shared/utils'

export interface ProjectSelectOption {
  name?: string
  type?: ComponentProps<typeof ProjectTypeBadge>['type']
}

export interface ProjectSelectProps {
  data?: ProjectSelectOption[]
  value?: string
  onChange?: (value: string) => void
  label?: InputBaseProps['label']
  withAsterisk?: InputBaseProps['withAsterisk']
  inputProps?: Omit<
    InputBaseProps,
    | 'component'
    | 'type'
    | 'children'
    | 'rightSection'
    | 'rightSectionPointerEvents'
    | 'onClick'
  > & {
    onBlur?: () => void
  }
}

interface SelectedProjectDisplayProps {
  name?: string
  type?: ComponentProps<typeof ProjectTypeBadge>['type']
}

function SelectedProjectDisplay({
  name,
  type,
}: SelectedProjectDisplayProps) {
  return (
    <Group gap={6} wrap="nowrap">
      <Text
        title={name}
        size="sm"
        truncate
      >
        {name ?? ''}
      </Text>
      <ProjectTypeBadge
        type={type}
        flex="0 0 auto"
      />
    </Group>
  )
}

const EMPTY_OPTIONS: ProjectSelectOption[] = []

export function ProjectSelect({
  data = EMPTY_OPTIONS,
  value,
  onChange,
  label,
  withAsterisk = true,
  inputProps,
}: ProjectSelectProps) {
  const { t } = useTranslation()
  const [search, setSearch] = useState('')
  const combobox = useCombobox({
    onDropdownClose: () => {
      combobox.resetSelectedOption()
      combobox.focusTarget()
      setSearch('')
    },
    onDropdownOpen: () => {
      combobox.updateSelectedOptionIndex('active', { scrollIntoView: true })
      combobox.focusSearchInput()
    },
  })
  const restInputProps = inputProps

  const selectedProjectOption = data.find(option => option.name === value)
  const filteredOptions = filterByKeyword(data, search)

  return (
    <Combobox
      store={combobox}
      onOptionSubmit={(nextValue) => {
        onChange?.(nextValue)
        combobox.closeDropdown()
      }}
    >
      <Combobox.Target>
        <InputBase
          component="button"
          type="button"
          label={label ?? (
            <FieldHintLabel
              label={t('shared.projectSelect.project')}
              hint={t('shared.projectSelect.projectTooltip')}
            />
          )}
          withAsterisk={withAsterisk}
          {...restInputProps}
          onBlur={() => inputProps?.onBlur?.()}
          rightSection={<Combobox.Chevron />}
          rightSectionPointerEvents="none"
          onClick={() => combobox.toggleDropdown()}
        >
          {selectedProjectOption
            ? (
                <SelectedProjectDisplay
                  name={selectedProjectOption.name}
                  type={selectedProjectOption.type}
                />
              )
            : (
                <Text c="dimmed" size="sm">
                  {t('shared.projectSelect.projectPlaceholder')}
                </Text>
              )}
        </InputBase>
      </Combobox.Target>

      <Combobox.Dropdown>
        <Combobox.Search
          value={search}
          placeholder={t('shared.search')}
          onChange={(event) => {
            combobox.updateSelectedOptionIndex()
            setSearch(event.currentTarget.value)
          }}
        />
        <Combobox.Options>
          <ScrollArea.Autosize
            mah={220}
            type="auto"
            scrollbarSize="var(--combobox-padding)"
            offsetScrollbars="y"
          >
            {filteredOptions.length
              ? filteredOptions.map((option) => {
                  const isSelected = option.name === value

                  return (
                    <Combobox.Option
                      value={option.name as string}
                      key={option.name}
                      active={isSelected}
                      aria-selected={isSelected}
                      bg={isSelected ? 'var(--mantine-primary-color-light)' : undefined}
                      c={isSelected ? 'var(--mantine-primary-color-light-color)' : undefined}
                    >
                      <Group justify="space-between" gap="xs" wrap="nowrap">
                        <SelectedProjectDisplay name={option.name} type={option.type} />
                        {isSelected && <CheckIcon size={12} />}
                      </Group>
                    </Combobox.Option>
                  )
                })
              : <Combobox.Empty>{t('common.noResults')}</Combobox.Empty>}
          </ScrollArea.Autosize>
        </Combobox.Options>
      </Combobox.Dropdown>
    </Combobox>
  )
}

import {
  Combobox,
  Group,
  InputBase,
  type InputBaseProps,
  Text,
  useCombobox,
} from '@mantine/core'
import { type ComponentProps } from 'react'
import { useTranslation } from 'react-i18next'

import { ProjectTypeBadge } from '@/shared/components/badges/ProjectTypeBadge'
import { FieldHintLabel } from '@/shared/components/FieldHintLabel.tsx'

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
  const combobox = useCombobox()
  const restInputProps = inputProps

  const selectedProjectOption = data.find(option => option.name === value)

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
        <Combobox.Options>
          {data.map(option => (
            <Combobox.Option value={option.name as string} key={option.name}>
              <SelectedProjectDisplay name={option.name} type={option.type} />
            </Combobox.Option>
          ))}
        </Combobox.Options>
      </Combobox.Dropdown>
    </Combobox>
  )
}

import {
  Combobox,
  Group,
  InputBase,
  type InputBaseProps,
  Text,
  Tooltip,
  useCombobox,
} from '@mantine/core'
import { type ComponentProps } from 'react'
import { useTranslation } from 'react-i18next'

import IconQuestion from '@/assets/svgs/question.svg?react'
import { ProjectTypeBadge } from '@/shared/components/badges/ProjectTypeBadge'

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
            <Group
              component="span"
              gap={4}
              align="center"
              wrap="nowrap"
              style={{ display: 'inline-flex' }}
            >
              <span>{t('shared.projectSelect.project')}</span>
              <Tooltip label={t('shared.projectSelect.projectTooltip')}>
                <IconQuestion width={18} height={18} />
              </Tooltip>
            </Group>
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

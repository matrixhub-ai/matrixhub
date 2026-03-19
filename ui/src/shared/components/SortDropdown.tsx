import {
  Box,
  Popover,
  UnstyledButton,
} from '@mantine/core'
import {
  useId,
  useState,
} from 'react'
import { useTranslation } from 'react-i18next'

import SortDescIcon from '@/assets/svgs/sort-desc.svg?react'

import classes from './SortDropdown.module.css'

import type {
  CSSProperties,
  ReactNode,
} from 'react'

export type SortOrder = 'asc' | 'desc'

export interface SortDropdownOption {
  value: string
  label: string
  icon?: ReactNode
  disabled?: boolean
}

interface SortDropdownProps {
  fieldOptions: readonly SortDropdownOption[]
  fieldValue: string
  order: SortOrder
  orderLabel?: string
  refreshing?: boolean
  width?: number
  orderWidth?: number
  onFieldChange: (value: string) => void
  onToggleOrder: () => void
}

const DEFAULT_WIDTH = 204
const DEFAULT_ORDER_WIDTH = 80

export function SortDropdown({
  fieldOptions,
  fieldValue,
  order,
  orderLabel,
  refreshing = false,
  width = DEFAULT_WIDTH,
  orderWidth = DEFAULT_ORDER_WIDTH,
  onFieldChange,
  onToggleOrder,
}: SortDropdownProps) {
  const { t } = useTranslation()
  const [opened, setOpened] = useState(false)
  const menuId = useId()
  const resolvedOrderLabel = orderLabel
    ?? t(order === 'asc' ? 'shared.sortOrderAsc' : 'shared.sortOrderDesc')
  const fieldWidth = width - orderWidth
  const selectedOption = fieldOptions.find(option => option.value === fieldValue)
    ?? fieldOptions[0]
  const cssVariables = {
    '--sort-width': `${width}px`,
    '--sort-field-width': `${fieldWidth}px`,
    '--sort-order-width': `${orderWidth}px`,
  } as CSSProperties

  return (
    <Popover
      opened={opened}
      onChange={setOpened}
      width={width}
      position="bottom-start"
      offset={8}
      withArrow
      arrowSize={10}
      withinPortal={false}
    >
      <Popover.Target>
        <Box className={classes.anchor} style={cssVariables}>
          <Box className={classes.control}>
            <UnstyledButton
              type="button"
              className={classes.fieldButton}
              aria-controls={menuId}
              aria-expanded={opened}
              aria-haspopup="menu"
              onClick={() => {
                setOpened(current => !current)
              }}
            >
              <span className={classes.label}>
                {selectedOption?.label ?? ''}
              </span>
            </UnstyledButton>

            <UnstyledButton
              type="button"
              className={classes.orderButton}
              onClick={() => {
                setOpened(false)
                onToggleOrder()
              }}
            >
              <span className={classes.label}>{resolvedOrderLabel}</span>
              <span
                className={classes.orderIcon}
                data-order={order}
                data-refreshing={refreshing}
              >
                <SortDescIcon width={16} height={16} />
              </span>
            </UnstyledButton>
          </Box>
        </Box>
      </Popover.Target>

      <Popover.Dropdown className={classes.dropdown} id={menuId}>
        <Box role="menu">
          {fieldOptions.map(option => (
            <UnstyledButton
              key={option.value}
              type="button"
              role="menuitemradio"
              className={classes.menuItem}
              tabIndex={option.disabled ? -1 : 0}
              data-selected={option.value === fieldValue}
              data-disabled={option.disabled}
              aria-checked={option.value === fieldValue}
              aria-disabled={option.disabled}
              onClick={() => {
                if (option.disabled) {
                  return
                }

                setOpened(false)

                if (option.value !== fieldValue) {
                  onFieldChange(option.value)
                }
              }}
            >
              {option.icon && (
                <span className={classes.menuItemIcon}>
                  {option.icon}
                </span>
              )}
              <span className={classes.menuItemLabel}>{option.label}</span>
            </UnstyledButton>
          ))}
        </Box>
      </Popover.Dropdown>
    </Popover>
  )
}

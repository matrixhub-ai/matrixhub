import {
  Group,
  Radio,
} from '@mantine/core'
import { DatePickerInput } from '@mantine/dates'
import dayjs from 'dayjs'
import { useTranslation } from 'react-i18next'

const defaultExpireAt = () => dayjs().add(1, 'day').format('YYYY-MM-DD')

interface ExpireAtFieldProps {
  value: string
  onChange: (value: string) => void
  error?: string
}

export function ExpireAtField({
  value,
  onChange,
  error,
}: ExpireAtFieldProps) {
  const { t } = useTranslation()
  const validityValue = value ? 'custom' : 'never'

  const handleValidityChange = (next: string) => {
    if (next === 'never') {
      onChange('')
    } else {
      onChange(defaultExpireAt())
    }
  }

  return (
    <>
      <Radio.Group
        label={t('profile.validity')}
        name="validity"
        value={validityValue}
        onChange={handleValidityChange}
      >
        <Group mt="xs">
          <Radio
            value="never"
            label={t('profile.expireNever')}
          />
          <Radio
            value="custom"
            label={t('profile.expireCustom')}
          />
        </Group>
      </Radio.Group>

      {validityValue === 'custom' && (
        <DatePickerInput
          label={t('profile.expireTime')}
          value={value}
          minDate={defaultExpireAt()}
          valueFormat="YYYY-MM-DD"
          highlightToday
          required
          onChange={next => onChange(next ?? '')}
          error={error}
        />
      )}
    </>
  )
}

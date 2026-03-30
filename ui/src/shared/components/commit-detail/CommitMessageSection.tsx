import {
  rem,
  Stack,
  Text,
} from '@mantine/core'

import { parseConventionalCommit } from '@/shared/utils'

interface CommitMessageSectionProps {
  message?: string
}

export function CommitMessageSection({ message }: CommitMessageSectionProps) {
  // TODO: some npm packages like `conventional-commits-parser` can parse the commit message into structured data,
  //  But it runs in Node environment and cannot be directly used in the browser. We may need to find a similar package that supports browser environment
  const parsedMessage = parseConventionalCommit(message ?? '')

  return (
    <Stack p="sm" gap="xs">
      <Text fw={600} size="sm" lh={rem(20)}>
        {parsedMessage?.header || '-'}
      </Text>
      {parsedMessage.body ? <Text size="sm" lh={rem(20)} style={{ whiteSpace: 'pre-wrap' }}>{parsedMessage.body}</Text> : null}
      {parsedMessage.footer ? <Text size="sm" lh={rem(20)}>{parsedMessage.footer}</Text> : null}
    </Stack>
  )
}

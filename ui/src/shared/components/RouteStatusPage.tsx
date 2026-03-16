import {
  Button,
  Center,
  Stack,
  Text,
  Title,
} from '@mantine/core'
import { Link } from '@tanstack/react-router'

interface RouteStatusPageProps {
  code: 403 | 404
  title?: string
  description?: string
  fullScreen?: boolean
}

export function RouteStatusPage({
  code,
  title,
  description,
  fullScreen = false,
}: RouteStatusPageProps) {
  const defaultDescriptions = {
    403: 'You do not have permission to access this page.',
    404: 'The page you are looking for does not exist or has been moved.',
  }

  const defaultTitles = {
    403: 'Access Denied',
    404: 'Page Not Found',
  }

  return (
    <Center h={fullScreen ? '100vh' : '100%'}>
      <Stack align="center" gap="md">
        <Title
          order={1}
          style={{
            fontSize: 96,
            lineHeight: 1,
            color: '#DEE2E6',
          }}
        >
          {code}
        </Title>
        <Title order={2}>
          {title ?? defaultTitles[code]}
        </Title>
        <Text c="dimmed" size="lg" ta="center" maw={400}>
          {description ?? defaultDescriptions[code]}
        </Text>
        <Button component={Link} to="/" mt="md">
          Back to Home
        </Button>
      </Stack>
    </Center>
  )
}

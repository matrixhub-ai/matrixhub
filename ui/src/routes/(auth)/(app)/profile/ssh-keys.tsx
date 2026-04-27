import { createFileRoute } from '@tanstack/react-router'

import { SshKeysPage } from '@/features/profile/pages/SshKeysPage'
import { sshKeysQueryOptions } from '@/features/profile/profile.query'

export const Route = createFileRoute('/(auth)/(app)/profile/ssh-keys')({
  loader: ({ context }) => {
    context.queryClient.prefetchQuery(sshKeysQueryOptions())
  },
  component: SshKeysPage,
})

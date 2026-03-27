import {
  Box, Collapse, Group, Text, UnstyledButton,
} from '@mantine/core'
import { useDisclosure } from '@mantine/hooks'
import { IconChevronDown } from '@tabler/icons-react'
import { getRouteApi } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'

import { splitFilterCsv, useModels } from '@/features/models/models.query.ts'
import { ModelCard } from '@/shared/components/resource-card/ModelCard.tsx'
import { ResourceCardGrid } from '@/shared/components/ResourceCardGrid'

const { useSearch } = getRouteApi('/(auth)/(app)/models/')

export function HotModelList() {
  const { t } = useTranslation()
  const search = useSearch()

  const [opened, { toggle }] = useDisclosure(false)

  const {
    data: {
      items = [],
    } = {},
    isLoading,
    isPending,
  } = useModels({
    project: search.project,
    labels: splitFilterCsv(search.task ?? search.library),
    page: 1,
    pageSize: -1,
    popular: true,
  })

  const cardElements = items.map((model) => {
    const projectId = model.project?.trim() ?? '-'
    const modelName = model.name?.trim() ?? '-'

    return <ModelCard key={`${projectId}/${modelName}`} model={model} />
  })

  return (
    <Box>
      <Group justify="space-between">
        <Text fz="md" fw={600} lh="20px" mb="sm">
          {t('model.list.recommend') }
        </Text>
        {cardElements.length > 4 && (
          <UnstyledButton onClick={toggle}>
            <Group gap={8}>
              <Text fz="sm" c="gray.5">
                {opened ? t('model.list.collapse') : t('model.list.viewMore')}
              </Text>
              <IconChevronDown
                size={16}
                color="var(--mantine-color-gray-5)"
                style={{
                  transform: opened ? 'rotate(180deg)' : 'rotate(0deg)',
                }}
              />
            </Group>
          </UnstyledButton>
        )}
      </Group>

      <Box miw={780} maw={1380}>
        <ResourceCardGrid loading={isLoading || isPending}>
          {cardElements.slice(0, 4)}
        </ResourceCardGrid>
        <Collapse in={opened}>
          <Box pt="lg">
            <ResourceCardGrid>
              {cardElements.slice(4)}
            </ResourceCardGrid>
          </Box>
        </Collapse>
      </Box>
    </Box>
  )
}

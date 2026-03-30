import {
  Breadcrumbs,
  Group,
  Text,
} from '@mantine/core'
import { type LinkComponentProps } from '@tanstack/react-router'

import AnchorLink from '@/shared/components/AnchorLink.tsx'
import { CopyValueButton } from '@/shared/components/CopyValueButton.tsx'

export interface ModelPathBreadcrumbsProps {
  name: string
  treePath?: string
  getPathLinkProps: (path: string) => LinkComponentProps
  copyValue?: string
}

export function PathBreadcrumbs({
  name,
  treePath,
  getPathLinkProps,
  copyValue,
}: ModelPathBreadcrumbsProps) {
  const pathSegments = treePath?.split('/').filter(Boolean) ?? []
  const rootLinkProps = getPathLinkProps('')

  return (
    <Breadcrumbs separator="/" separatorMargin="xs">
      <AnchorLink
        c="gray.9"
        underline="hover"
        {...rootLinkProps}
      >
        {name}
      </AnchorLink>

      {pathSegments.map((segment, index) => {
        const pathToSegment = pathSegments.slice(0, index + 1).join('/')
        const pathLinkProps = getPathLinkProps?.(pathToSegment)
        const isLastSegment = index === pathSegments.length - 1
        const copiedValue = copyValue ?? segment

        if (!pathLinkProps || isLastSegment) {
          return (
            <Group key={pathToSegment} gap={4} wrap="nowrap">
              <Text c="gray.9">
                {segment}
              </Text>
              {isLastSegment && copiedValue && (
                <CopyValueButton value={copiedValue} iconSize={14} />
              )}
            </Group>
          )
        }

        return (
          <AnchorLink
            c="gray.9"
            underline="hover"
            key={pathToSegment}
            {...pathLinkProps}
          >
            {segment}
          </AnchorLink>
        )
      })}
    </Breadcrumbs>
  )
}

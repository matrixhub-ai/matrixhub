import {
  Breadcrumbs,
  Text,
} from '@mantine/core'
import { type LinkComponentProps } from '@tanstack/react-router'

import AnchorLink from '@/shared/components/AnchorLink.tsx'

export interface ModelPathBreadcrumbsProps {
  name: string
  treePath?: string
  getPathLinkProps: (path: string) => LinkComponentProps
}

export function PathBreadcrumbs({
  name,
  treePath,
  getPathLinkProps,
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

        if (!pathLinkProps || index === pathSegments.length - 1) {
          return (
            <Text key={pathToSegment} c="gray.9">
              {segment}
            </Text>
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

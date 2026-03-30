import { Text } from '@mantine/core'
import { FileType } from '@matrixhub/api-ts/v1alpha1/model.pb'
import {
  IconFile,
  IconFileDescription,
  IconFileSettings,
  IconFolder,
} from '@tabler/icons-react'
import { Link, type LinkComponentProps } from '@tanstack/react-router'
import { useMemo } from 'react'
import { useTranslation } from 'react-i18next'

import AnchorLink from '@/shared/components/AnchorLink'
import { DataTable } from '@/shared/components/DataTable'
import { formatRelativeTime } from '@/shared/utils/date'
import { formatStorageSize } from '@/shared/utils/format'

import classes from './RepoFileTree.module.css'

import type { File as RepoFile } from '@matrixhub/api-ts/v1alpha1/model.pb'
import type { MRT_ColumnDef } from 'mantine-react-table'

type LinkPropsBuilder<T> = (value: T) => LinkComponentProps

interface RepoFileTreeMeta {
  buildTreeLink: LinkPropsBuilder<string>
  buildBlobLink: LinkPropsBuilder<string>
}

interface RepoFileTreeProps {
  files: RepoFile[]
  latestCommit?: RepoFile['commit']
  buildTreeLink: LinkPropsBuilder<string>
  buildBlobLink: LinkPropsBuilder<string>
  buildCommitLink?: LinkPropsBuilder<string>
  isLoading?: boolean
}

type FileCellProps = Parameters<NonNullable<MRT_ColumnDef<RepoFile>['Cell']>>[0]

function FileNameCell({
  row, table,
}: FileCellProps) {
  const file = row.original
  const isDir = file.type === FileType.DIR
  const filePath = file.path ?? file.name ?? ''
  const fileName = file.name ?? ''
  const {
    buildTreeLink, buildBlobLink,
  } = table.options.meta as RepoFileTreeMeta
  const linkProps = isDir ? buildTreeLink(filePath) : buildBlobLink(filePath)

  return (
    <div className={classes.nameCell}>
      <span className={classes.nameIcon}>
        <FileIcon file={file} />
      </span>
      <Link className={classes.fileLink} {...linkProps}>
        {fileName}
      </Link>
    </div>
  )
}

function FileSizeCell({ row }: FileCellProps) {
  if (row.original.type === FileType.DIR) {
    return null
  }

  return (
    <Text size="sm" c="gray.7">
      {formatStorageSize(row.original.size)}
    </Text>
  )
}

function FileCommitCell({ row }: FileCellProps) {
  return (
    <Text size="sm" c="gray.7" truncate>
      {row.original.commit?.message ?? ''}
    </Text>
  )
}

function FileDateCell({ row }: FileCellProps) {
  return (
    <Text size="sm" c="gray.7">
      {formatRelativeTime(row.original.commit?.createdAt)}
    </Text>
  )
}

function FileActionCell({ row }: FileCellProps) {
  const { t } = useTranslation()
  const file = row.original

  if (file.type === FileType.DIR) {
    return null
  }

  if (file.url) {
    return (
      <a
        className={classes.downloadLink}
        href={file.url}
        download
        onClick={e => e.stopPropagation()}
      >
        {t('common.fileTree.download')}
      </a>
    )
  }

  return (
    <Text size="sm" c="gray.6">
      {t('common.fileTree.download')}
    </Text>
  )
}

function FileIcon({ file }: { file: RepoFile }) {
  const isDir = file.type === FileType.DIR
  const name = file.name?.toLocaleLowerCase() ?? ''

  if (isDir) {
    return <IconFolder size={20} color="var(--mantine-color-cyan-6)" />
  }

  if (name === 'readme' || name === 'readme.md') {
    return <IconFileDescription size={20} color="var(--mantine-color-gray-6)" />
  }

  if (
    name.endsWith('.json')
    || name.endsWith('.yaml')
    || name.endsWith('.yml')
    || name.endsWith('.toml')
  ) {
    return <IconFileSettings size={20} color="var(--mantine-color-gray-6)" />
  }

  return <IconFile size={20} color="var(--mantine-color-gray-6)" />
}

export function RepoFileTree({
  files,
  latestCommit,
  buildTreeLink,
  buildBlobLink,
  buildCommitLink,
  isLoading,
}: RepoFileTreeProps) {
  const { t } = useTranslation()

  const sortedFiles = useMemo(() => sortFiles(files), [files])

  const columns: MRT_ColumnDef<RepoFile>[] = [
    {
      id: 'name',
      Cell: FileNameCell,
      header: '',
      size: 185,
      grow: false,
    },
    {
      id: 'size',
      header: '',
      Cell: FileSizeCell,
      size: 185,
      grow: false,
    },
    {
      id: 'commit',
      header: '',
      Cell: FileCommitCell,
      size: 360,
    },
    {
      id: 'date',
      header: '',
      Cell: FileDateCell,
      size: 185,
      grow: false,
    },
    {
      id: 'action',
      header: '',
      Cell: FileActionCell,
      size: 185,
      grow: false,
    },
  ]

  return (
    <div className={classes.wrapper}>
      {latestCommit && (
        <div className={classes.caption}>
          <div className={classes.captionLeft}>
            <span className={classes.captionAuthor}>{latestCommit.authorName}</span>
            <span className={classes.captionMessage}>{latestCommit.message}</span>
            {latestCommit.id && buildCommitLink && (
              <AnchorLink
                className={classes.captionHash}
                size="sm"
                underline="hover"
                c="gray.6"
                {...buildCommitLink(latestCommit.id)}
              >
                {latestCommit.id.slice(0, 8)}
              </AnchorLink>
            )}
            {latestCommit.id && !buildCommitLink && (
              <span className={classes.captionHash}>{latestCommit.id.slice(0, 8)}</span>
            )}
          </div>
          <span className={classes.captionTime}>
            {formatRelativeTime(latestCommit.createdAt)}
          </span>
        </div>
      )}

      <DataTable
        data={sortedFiles}
        columns={columns}
        loading={isLoading}
        emptyTitle={t('common.noResults')}
        hideTableHead
        tableOptions={{
          meta: {
            buildTreeLink,
            buildBlobLink,
          },
          mantineTableBodyCellProps: {
            style: {
              height: 44,
              padding: '0 12px',
            },
          },
        }}
      />
    </div>
  )
}

function sortFiles(files: RepoFile[]): RepoFile[] {
  return [...files].sort((a, b) => {
    const aIsDir = a.type === FileType.DIR ? 0 : 1
    const bIsDir = b.type === FileType.DIR ? 0 : 1

    if (aIsDir !== bIsDir) {
      return aIsDir - bIsDir
    }

    return (a.name ?? '').localeCompare(b.name ?? '')
  })
}

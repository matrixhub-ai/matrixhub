import { ProjectRoleType } from '@matrixhub/api-ts/v1alpha1/role.pb'
import { useTranslation } from 'react-i18next'

const ROLE_LEVEL: Record<string, number> = {
  [ProjectRoleType.ROLE_TYPE_PROJECT_ADMIN]: 3,
  [ProjectRoleType.ROLE_TYPE_PROJECT_EDITOR]: 2,
  [ProjectRoleType.ROLE_TYPE_PROJECT_VIEWER]: 1,
}

export const useProjectRoleOptions = (currentRole?: ProjectRoleType) => {
  const { t } = useTranslation()
  const maxLevel = currentRole ? (ROLE_LEVEL[currentRole] ?? 0) : Infinity

  return [
    {
      value: ProjectRoleType.ROLE_TYPE_PROJECT_ADMIN,
      label: t('projects.detail.membersPage.role.admin'),
    },
    {
      value: ProjectRoleType.ROLE_TYPE_PROJECT_EDITOR,
      label: t('projects.detail.membersPage.role.editor'),
    },
    {
      value: ProjectRoleType.ROLE_TYPE_PROJECT_VIEWER,
      label: t('projects.detail.membersPage.role.viewer'),
    },
  ].filter(opt => (ROLE_LEVEL[opt.value] ?? 0) <= maxLevel)
}

import { ProjectRoleType } from '@matrixhub/api-ts/v1alpha1/role.pb'
import { useTranslation } from 'react-i18next'

export const useProjectRoleOptions = () => {
  const { t } = useTranslation()

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
  ]
}

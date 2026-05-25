import { useSuspenseQuery } from '@tanstack/react-query'

import { RobotAccountForm } from '../components/RobotAccountForm'
import { robotAccountDetailQueryOptions } from '../robot.query'
import { getRobotAccountFormDefaults } from '../robot.schema'

interface RobotAccountEditPageProps {
  robotId: number
}

export function RobotAccountEditPage({
  robotId,
}: RobotAccountEditPageProps) {
  const { data } = useSuspenseQuery(robotAccountDetailQueryOptions(robotId))

  return (
    <RobotAccountForm
      mode="edit"
      robot={data}
      defaultValues={getRobotAccountFormDefaults(data)}
    />
  )
}

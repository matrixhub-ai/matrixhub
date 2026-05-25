import { RobotAccountForm } from '../components/RobotAccountForm'
import { getRobotAccountFormDefaults } from '../robot.schema'

export function RobotAccountCreatePage() {
  return (
    <RobotAccountForm
      mode="create"
      defaultValues={getRobotAccountFormDefaults()}
    />
  )
}

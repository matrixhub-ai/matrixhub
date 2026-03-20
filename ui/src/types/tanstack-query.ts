/** Shared meta shape for query and mutation notification behavior. */
export interface NotificationMeta {
  /** Notification message shown on success */
  successMessage?: string
  /** Notification title shown on error */
  errorMessage?: string
  /** Skip all notifications for this query/mutation */
  skipNotification?: boolean
  /** Query keys to invalidate on mutation success */
  invalidates?: readonly unknown[][]
}

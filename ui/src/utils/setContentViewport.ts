const restorationIdKey = 'data-scroll-restoration-id'

const adminDataSelector = 'admin-content-scroll'
const contentDataSelector = 'content-scroll'

export const adminContentViewportSelector = `[${restorationIdKey}="${adminDataSelector}"]`
export const contentViewportSelector = `[${restorationIdKey}="${contentDataSelector}"]`

export function setAdminContentViewport(viewport: HTMLDivElement | null) {
  viewport?.setAttribute(restorationIdKey, adminDataSelector)
}

export function setContentViewport(viewport: HTMLDivElement | null) {
  viewport?.setAttribute(restorationIdKey, contentDataSelector)
}

import { z } from 'zod'

import i18n from '@/i18n'

const NAME_REGEX = /^[A-Za-z0-9](?:[A-Za-z0-9-]*[A-Za-z0-9])$/

// ---------------------------------------------------------------------------
// Field-level schemas — used on each <form.Field validators={{ onBlur }}>
// ---------------------------------------------------------------------------

export const projectNameSchema = z
  .string()
  .trim()
  .superRefine((val, ctx) => {
    if (!val) {
      ctx.addIssue({
        code: 'custom',
        message: i18n.t('projects.validation.nameRequired'),
      })

      return
    }

    if (val.length < 2) {
      ctx.addIssue({
        code: 'custom',
        message: i18n.t('projects.validation.nameMinLength'),
      })

      return
    }

    if (!NAME_REGEX.test(val)) {
      ctx.addIssue({
        code: 'custom',
        message: i18n.t('projects.validation.nameFormat'),
      })

      return
    }
  })

export const registryIdSchema = z
  .number().optional()
  .superRefine((val, ctx) => {
    if (val == null) {
      ctx.addIssue({
        code: 'custom',
        message: i18n.t('projects.validation.registryRequired'),
      })
    }
  })

export const organizationSchema = z
  .string()
  .trim()
  .optional()
  .superRefine((val, ctx) => {
    if (!val) {
      ctx.addIssue({
        code: 'custom',
        message: i18n.t('projects.validation.organizationRequired'),
      })
    }
  })

// ---------------------------------------------------------------------------
// Form-level schema — used on `useForm({ validators: { onSubmit } })`
//
// Field validators only run once a field is edited, so a proxy project whose
// registry select is never touched would otherwise submit an empty
// `registryId` and fail on the backend foreign key. This re-checks the
// proxy-only fields at submit time, and only while proxy is enabled.
// ---------------------------------------------------------------------------

export const createProjectSchema = z
  .object({
    name: projectNameSchema,
    isPublic: z.boolean(),
    enabledProxy: z.boolean(),
    // `z.union` with `undefined` rather than `.optional()`: the form value
    // always carries these keys, so the schema input must too.
    registryId: z.union([z.number(), z.undefined()]),
    organization: z.union([z.string().trim(), z.undefined()]),
  })
  .superRefine((val, ctx) => {
    if (!val.enabledProxy) {
      return
    }

    if (val.registryId == null) {
      ctx.addIssue({
        code: 'custom',
        path: ['registryId'],
        message: i18n.t('projects.validation.registryRequired'),
      })
    }

    if (!val.organization) {
      ctx.addIssue({
        code: 'custom',
        path: ['organization'],
        message: i18n.t('projects.validation.organizationRequired'),
      })
    }
  })

export interface CreateProjectInput {
  name: string
  isPublic: boolean
  enabledProxy: boolean
  registryId?: number
  organization?: string
}

import { RegistryType } from '@matrixhub/api-ts/v1alpha1/registry.pb'
import { z } from 'zod'

import i18n from '@/i18n'

import type {
  CreateRegistryRequest,
  PingRegistryRequest,
  Registry,
  RegistryBasicCredential,
  UpdateRegistryRequest,
} from '@matrixhub/api-ts/v1alpha1/registry.pb'

export const REGISTRY_NAME_MAX_LENGTH = 64
export const REGISTRY_NAME_MIN_LENGTH = 2
export const REGISTRY_DESCRIPTION_MAX_LENGTH = 50

export interface RegistryFormValues {
  type: RegistryType
  name: string
  description: string
  url: string
  username: string
  password: string
  verifyRemoteCert: boolean
}

export const defaultRegistryFormValues: RegistryFormValues = {
  type: RegistryType.REGISTRY_TYPE_HUGGINGFACE,
  name: '',
  description: '',
  url: '',
  username: '',
  password: '',
  verifyRemoteCert: true,
}

/* Name validation:
No spaces (including full-width/half-width) at the start, end, or within; auto-trimmed;
length 2-64 characters */
export function sanitizeRegistryName(value: string) {
  return value.replace(/[\s\u3000]+/gu, '')
}

export const registryNameSchema = z
  .string()
  .superRefine((value, ctx) => {
    const sanitizedValue = sanitizeRegistryName(value)

    if (
      sanitizedValue.length < REGISTRY_NAME_MIN_LENGTH
      || sanitizedValue.length > REGISTRY_NAME_MAX_LENGTH
    ) {
      ctx.addIssue({
        code: 'custom',
        message: i18n.t('routes.admin.registries.validation.nameLength', {
          min: REGISTRY_NAME_MIN_LENGTH,
          max: REGISTRY_NAME_MAX_LENGTH,
        }),
      })
    }
  })

export const registryDescriptionSchema = z
  .string()
  .trim()
  .max(REGISTRY_DESCRIPTION_MAX_LENGTH, {
    error: i18n.t('common.validation.maxLength', {
      field: i18n.t('routes.admin.registries.form.description'),
      max: REGISTRY_DESCRIPTION_MAX_LENGTH,
    }),
  })

export const registryUrlSchema = z
  .string()
  .trim()
  .superRefine((value, ctx) => {
    if (!value) {
      ctx.addIssue({
        code: 'custom',
        message: i18n.t('routes.admin.registries.validation.urlRequired'),
      })

      return
    }

    try {
      const parsedUrl = new URL(value)

      if (parsedUrl.protocol !== 'http:' && parsedUrl.protocol !== 'https:') {
        throw new Error('unsupported protocol')
      }
    } catch {
      ctx.addIssue({
        code: 'custom',
        message: i18n.t('routes.admin.registries.validation.urlInvalid'),
      })
    }
  })

export const createRegistryFormSchema = z.object({
  type: z.enum([RegistryType.REGISTRY_TYPE_HUGGINGFACE]),
  name: registryNameSchema,
  description: registryDescriptionSchema,
  url: registryUrlSchema,
  username: z.string().trim(),
  password: z.string().trim(),
  verifyRemoteCert: z.boolean(),
})

export const editRegistryFormSchema = createRegistryFormSchema

export function getRegistryFormValues(registry?: Registry | null): RegistryFormValues {
  return {
    ...defaultRegistryFormValues,
    type: registry?.type ?? RegistryType.REGISTRY_TYPE_HUGGINGFACE,
    name: sanitizeRegistryName(registry?.name ?? ''),
    description: registry?.description ?? '',
    url: registry?.url ?? '',
    username: registry?.basic?.username ?? '',
    password: registry?.basic?.password ?? '',
    verifyRemoteCert: !registry?.insecure,
  }
}

function getBasicCredential(values: RegistryFormValues): RegistryBasicCredential | undefined {
  const username = values.username.trim()
  const password = values.password.trim()

  if (!username && !password) {
    return undefined
  }

  return {
    username,
    password,
  }
}

export function buildCreateRegistryRequest(values: RegistryFormValues): CreateRegistryRequest {
  const basic = getBasicCredential(values)

  return {
    name: sanitizeRegistryName(values.name),
    description: values.description.trim(),
    type: values.type,
    url: values.url.trim(),
    insecure: !values.verifyRemoteCert,
    ...(basic ? { basic } : {}),
  }
}

export function buildUpdateRegistryRequest(
  registryId: number,
  values: RegistryFormValues,
): UpdateRegistryRequest {
  const basic = getBasicCredential(values)

  return {
    id: registryId,
    name: sanitizeRegistryName(values.name),
    description: values.description.trim(),
    url: values.url.trim(),
    insecure: !values.verifyRemoteCert,
    ...(basic ? { basic } : {}),
  }
}

export function buildPingRegistryRequest(values: RegistryFormValues): PingRegistryRequest {
  const basic = getBasicCredential(values)

  return {
    type: values.type,
    url: values.url.trim(),
    insecure: !values.verifyRemoteCert,
    ...(basic ? { basic } : {}),
  }
}

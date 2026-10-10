import { z } from 'zod'

export const securityAuditActorSchema = z.object({
  type: z.literal('user'),
  payload: z.object({ name: z.string().min(1).max(200) }),
})

export const securitySearchSchema = z.object({
  revision: z.string().trim().min(1).max(120).optional().default('main').catch('main'),
})
export const securityRevisionSchema = z.string().trim().min(1).max(120)
export const securityPolicySchema = z.object({
  block_severity: z.enum(['medium', 'high']),
  on_pending: z.enum(['block', 'allow']),
  on_failure: z.enum(['block', 'allow']),
})

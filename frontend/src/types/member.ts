import { z } from 'zod';
import { locales } from '@/i18n/config';

/** Member as returned by the auth and profile endpoints (plan section 4.2). */
export const memberSchema = z.object({
  id: z.string(),
  nickname: z.string(),
  avatar_url: z.string().nullish(),
  preferred_language: z.enum(locales).catch('ja')
});

export type Member = z.infer<typeof memberSchema>;

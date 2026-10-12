import { useMutation } from '@tanstack/react-query';
import { z } from 'zod';
import { locales } from '@/i18n/config';
import { api } from '@/lib/api-client';

export const translateInputSchema = z.object({
  text: z.string().min(1).max(2000),
  // `auto` lets the server detect the language from the script, for text whose language is not recorded.
  source_lang: z.enum([...locales, 'auto']),
  target_lang: z.enum(locales)
});
export type TranslateInput = z.infer<typeof translateInputSchema>;

export const translateResponseSchema = z.object({
  translated_text: z.string(),
  provider: z.string(),
  cached: z.boolean(),
  source_lang: z.string()
});
export type TranslateResponse = z.infer<typeof translateResponseSchema>;

// The translation provider is called by the Go API only; the browser never talks to it.
const translate = async (data: TranslateInput): Promise<TranslateResponse> => {
  const response = await api.post('/translate', translateInputSchema.parse(data));
  return translateResponseSchema.parse(response.data);
};

export const useTranslate = () => useMutation({ mutationFn: translate });

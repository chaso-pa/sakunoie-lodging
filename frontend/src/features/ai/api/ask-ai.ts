import { useMutation } from '@tanstack/react-query';
import { z } from 'zod';
import { aiFiltersSchema } from '@/features/ai/api/get-ai-spots';
import { locales } from '@/i18n/config';
import { api } from '@/lib/api-client';

export const aiMessageMaxLength = 3000;

export const askAiInputSchema = aiFiltersSchema.extend({
  message: z.string().trim().min(1).max(aiMessageMaxLength),
  language: z.enum(locales)
});
export type AskAiInput = z.infer<typeof askAiInputSchema>;

export const askAiResponseSchema = z.object({
  /** Plain text. It is rendered as text, never as HTML. */
  answer: z.string(),
  model: z.string()
});
export type AskAiResponse = z.infer<typeof askAiResponseSchema>;

// The Go API calls the model provider; no key or provider URL ever reaches the browser.
const askAi = async (data: AskAiInput): Promise<AskAiResponse> => {
  const response = await api.post('/ai/plan', askAiInputSchema.parse(data));
  return askAiResponseSchema.parse(response.data);
};

export const useAskAi = () => useMutation({ mutationFn: askAi });

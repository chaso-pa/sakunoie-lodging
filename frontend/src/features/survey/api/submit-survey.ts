import { useMutation } from '@tanstack/react-query';
import type { SurveyPayload } from '@/features/survey/utils/build-payload';
import { api } from '@/lib/api-client';

/** `POST /surveys` → 202 with no body. Anonymous: no token, no identifiers. */
export const submitSurvey = (payload: SurveyPayload) => api.post<void>('/surveys', payload).then(() => undefined);

export const useSubmitSurvey = () => useMutation({ mutationFn: submitSurvey });

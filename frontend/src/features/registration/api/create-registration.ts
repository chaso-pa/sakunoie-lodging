import { useMutation } from '@tanstack/react-query';
import type { CreateRegistrationResult } from '@/features/registration/types/api';
import { type CreateRegistrationPayload, createRegistrationPayloadSchema } from '@/features/registration/types/schemas';
import { ApiError, api, toApiError } from '@/lib/api-client';

const NETWORK_RETRIES = 2;
const sleep = (ms: number) => new Promise((resolve) => setTimeout(resolve, ms));

/**
 * Creates a registration. Network failures and timeouts are retried with the **same**
 * Idempotency-Key, so a response lost after the server committed is recovered as a replay.
 * The caller owns the key (wizard store) and never mints a new one on failure.
 */
export const createRegistration = async ({
  payload,
  idempotencyKey
}: {
  payload: CreateRegistrationPayload;
  idempotencyKey: string;
}): Promise<CreateRegistrationResult> => {
  const body = createRegistrationPayloadSchema.parse(payload);
  for (let attempt = 0; ; attempt++) {
    try {
      const response = await api.post<CreateRegistrationResult>('/guest-registrations', body, {
        headers: { 'Idempotency-Key': idempotencyKey }
      });
      return response.data;
    } catch (error) {
      const apiError = toApiError(error);
      if (!(apiError instanceof ApiError) || !apiError.isNetworkError || attempt >= NETWORK_RETRIES) throw apiError;
      await sleep(1000 * (attempt + 1));
    }
  }
};

export const useCreateRegistration = () => useMutation({ mutationFn: createRegistration });

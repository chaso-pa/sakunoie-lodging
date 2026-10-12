import { QueryClient } from '@tanstack/react-query';
import { getApiErrorStatus } from '@/lib/api-client';

export const makeQueryClient = () =>
  new QueryClient({
    defaultOptions: {
      queries: {
        staleTime: 60 * 1000,
        refetchOnWindowFocus: false,
        // Client errors (401, 403, 404, 429...) will not succeed on retry.
        retry: (failureCount, error) => {
          const status = getApiErrorStatus(error);
          if (status !== undefined && status >= 400 && status < 500) {
            return false;
          }
          return failureCount < 2;
        }
      }
    }
  });

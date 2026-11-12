import { QueryClient } from '@tanstack/react-query';
import { ApiError } from '@/lib/api-client';

export const makeQueryClient = () =>
  new QueryClient({
    defaultOptions: {
      queries: {
        refetchOnWindowFocus: false,
        // Retry only transient failures; 4xx answers (expired token, not found) are final.
        retry: (failureCount, error) =>
          failureCount < 2 && (!(error instanceof ApiError) || error.isNetworkError || error.status >= 500)
      },
      mutations: { retry: false }
    }
  });

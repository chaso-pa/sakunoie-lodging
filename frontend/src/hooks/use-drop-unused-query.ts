import { hashKey, type QueryKey, useQueryClient } from '@tanstack/react-query';
import { useEffect } from 'react';

/**
 * Removes a query from the cache as soon as this component stops using it (its key changes, or the
 * component unmounts), when `shouldDrop` says its data may not be kept. Google's terms allow its place
 * content to be shown, not cached, so such data must not wait out the cache time for a later reuse.
 * While the component uses the query the data stays, and is not fetched again.
 */
export const useDropUnusedQuery = <TData>(queryKey: QueryKey, shouldDrop: (data: TData | undefined) => boolean) => {
  const queryClient = useQueryClient();
  const hash = hashKey(queryKey);
  useEffect(
    () => () => {
      const cache = queryClient.getQueryCache();
      const query = cache.get(hash);
      if (query && shouldDrop(query.state.data as TData | undefined)) {
        cache.remove(query);
      }
    },
    [queryClient, hash, shouldDrop]
  );
};

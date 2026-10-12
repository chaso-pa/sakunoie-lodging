'use client';

import { MantineProvider } from '@mantine/core';
import { Notifications } from '@mantine/notifications';
import { QueryClientProvider } from '@tanstack/react-query';
import { useState } from 'react';
import { makeQueryClient } from '@/lib/query-client';
import { theme } from '@/styles/theme';

export function Providers({ children }: { children: React.ReactNode }) {
  const [queryClient] = useState(makeQueryClient);

  return (
    <MantineProvider theme={theme} defaultColorScheme='light'>
      <QueryClientProvider client={queryClient}>
        <Notifications position='bottom-center' />
        {children}
      </QueryClientProvider>
    </MantineProvider>
  );
}

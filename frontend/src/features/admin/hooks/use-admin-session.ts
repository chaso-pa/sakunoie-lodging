'use client';

import { createContext, useContext } from 'react';
import type { AdminFacility } from '@/features/admin/types/api';

export type AdminSession = { uid: string; email: string | null; facilities: AdminFacility[] };

export const AdminSessionContext = createContext<AdminSession | null>(null);

/** The signed-in, authorized admin. Only available inside `AdminConsole`. */
export const useAdminSession = () => {
  const session = useContext(AdminSessionContext);
  if (!session) throw new Error('useAdminSession must be used inside AdminConsole');
  return session;
};

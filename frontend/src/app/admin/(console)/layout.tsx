import type { ReactNode } from 'react';
import { AdminConsole } from '@/features/admin';

/** Every page in this group requires a signed-in, authorized admin. */
export default function AdminConsoleLayout({ children }: { children: ReactNode }) {
  return <AdminConsole>{children}</AdminConsole>;
}

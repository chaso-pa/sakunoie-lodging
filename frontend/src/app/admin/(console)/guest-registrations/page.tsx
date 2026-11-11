import type { Metadata } from 'next';
import { Suspense } from 'react';
import { AdminLedgerView } from '@/features/admin';

export const metadata: Metadata = { title: '宿泊者名簿' };

export default function AdminRegistrationsPage() {
  return (
    <Suspense>
      <AdminLedgerView />
    </Suspense>
  );
}

import type { Metadata } from 'next';
import { AdminInvitationsView } from '@/features/admin';

export const metadata: Metadata = { title: '招待リンク' };

export default function AdminInvitationsPage() {
  return <AdminInvitationsView />;
}

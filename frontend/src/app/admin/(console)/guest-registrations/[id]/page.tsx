import type { Metadata } from 'next';
import { notFound } from 'next/navigation';
import { AdminRegistrationDetailView } from '@/features/admin';

export const metadata: Metadata = { title: '登録の詳細' };

const UUID_RE = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i;

export default async function AdminRegistrationPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = await params;
  if (!UUID_RE.test(id)) notFound();
  return <AdminRegistrationDetailView id={id} />;
}

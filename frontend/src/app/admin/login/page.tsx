import type { Metadata } from 'next';
import { AdminLoginView } from '@/features/admin';

export const metadata: Metadata = { title: 'ログイン' };

export default function AdminLoginPage() {
  return <AdminLoginView />;
}

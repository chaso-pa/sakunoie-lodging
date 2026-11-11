import type { Metadata } from 'next';
import { AdminSurveysView } from '@/features/admin';

export const metadata: Metadata = { title: 'アンケート集計' };

export default function AdminSurveysPage() {
  return <AdminSurveysView />;
}

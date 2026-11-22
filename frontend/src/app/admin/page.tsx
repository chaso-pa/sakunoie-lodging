import { redirect } from 'next/navigation';
import { paths } from '@/config/paths';

export default function AdminIndexPage() {
  redirect(paths.admin.registrations.getHref());
}

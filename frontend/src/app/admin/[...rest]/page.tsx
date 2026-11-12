import { notFound } from 'next/navigation';

/** Unknown /admin/* paths render admin/not-found.tsx instead of the global 404. */
export default function AdminCatchAll() {
  notFound();
}

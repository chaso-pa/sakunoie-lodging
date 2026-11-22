import { notFound } from 'next/navigation';

/** Unknown localized paths render `[locale]/not-found.tsx` inside the localized layout. */
export default function CatchAllPage() {
  notFound();
}

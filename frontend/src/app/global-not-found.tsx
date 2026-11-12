import type { Metadata } from 'next';
import { paths } from '@/config/paths';

export const metadata: Metadata = { title: '404 | 佐久の家 / Saku no Ie' };

/**
 * Root 404 for URLs outside `[locale]` (Next 16 `experimental.globalNotFound`).
 * Self-contained markup: it bypasses every layout, so it uses no Mantine or next-intl.
 */
export default function GlobalNotFound() {
  return (
    <html lang='ja'>
      <body
        style={{ margin: 0, fontFamily: 'system-ui, sans-serif', background: '#F7F4EE', color: '#292D2A', padding: 32 }}
      >
        <main>
          <h1 style={{ fontSize: 28 }}>ページが見つかりません / Page not found</h1>
          <p>
            <a href={paths.public.home.getHref()} style={{ color: '#934B32' }}>
              佐久の家 / Saku no Ie
            </a>
          </p>
        </main>
      </body>
    </html>
  );
}

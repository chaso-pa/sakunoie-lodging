import type { Metadata } from 'next';
import { getTranslations } from 'next-intl/server';
import classes from '@/app/(community)/layout.module.scss';
import { AuthGate } from '@/features/auth';

export async function generateMetadata(): Promise<Metadata> {
  const t = await getTranslations();
  return {
    title: `${t('auth.brand')}｜${t('common.siteName')}`,
    description: t('community.tagline'),
    robots: { index: false }
  };
}

/**
 * Shell for every community route. Everything inside AuthGate is rendered for signed-in members only,
 * so the community header and bottom navigation belong inside it, around `children`.
 */
export default function CommunityLayout({ children }: { children: React.ReactNode }) {
  return (
    <div className={classes.backdrop}>
      <div className={classes.app}>
        <AuthGate>{children}</AuthGate>
      </div>
    </div>
  );
}

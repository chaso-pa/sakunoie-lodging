import type { Metadata } from 'next';
import { getTranslations, setRequestLocale } from 'next-intl/server';
import { ManageView } from '@/features/registration';
import type { Locale } from '@/i18n/routing';

type PageProps = { params: Promise<{ locale: string }> };

export async function generateMetadata({ params }: PageProps): Promise<Metadata> {
  const locale = (await params).locale as Locale;
  const t = await getTranslations({ locale, namespace: 'manage' });
  return { title: t('metaTitle'), robots: { index: false, follow: false } };
}

export default async function RegistrationManagePage({ params }: PageProps) {
  setRequestLocale((await params).locale as Locale);
  return <ManageView />;
}

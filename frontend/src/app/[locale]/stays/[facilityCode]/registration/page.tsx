import type { Metadata } from 'next';
import { notFound } from 'next/navigation';
import { getTranslations, setRequestLocale } from 'next-intl/server';
import { RegistrationWizard } from '@/features/registration';
import type { Locale } from '@/i18n/routing';

type PageProps = { params: Promise<{ locale: string; facilityCode: string }> };

const FACILITY_CODE_PATTERN = /^[a-z0-9-]{1,64}$/;

export async function generateMetadata({ params }: PageProps): Promise<Metadata> {
  const locale = (await params).locale as Locale;
  const t = await getTranslations({ locale, namespace: 'registration' });
  return { title: t('metaTitle'), robots: { index: false, follow: false } };
}

export default async function RegistrationPage({ params }: PageProps) {
  const { locale, facilityCode } = await params;
  setRequestLocale(locale as Locale);
  if (!FACILITY_CODE_PATTERN.test(facilityCode)) notFound();
  return <RegistrationWizard facilityCode={facilityCode} />;
}

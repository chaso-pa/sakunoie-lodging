import { Container, Stack, Text, Title } from '@mantine/core';
import type { Metadata } from 'next';
import { getTranslations, setRequestLocale } from 'next-intl/server';
import { InfoBanner } from '@/components/InfoBanner';
import { contentFacilityCodes, defaultFacilityCode, getFacilityContent } from '@/config/site';
import { surveyDefinition } from '@/config/survey-definition';
import { SurveyForm } from '@/features/survey';
import type { Locale } from '@/i18n/routing';

type PageProps = {
  params: Promise<{ locale: string }>;
  searchParams: Promise<{ facility?: string | string[] }>;
};

export async function generateMetadata({ params }: PageProps): Promise<Metadata> {
  const locale = (await params).locale as Locale;
  const t = await getTranslations({ locale, namespace: 'survey' });
  return { title: t('metaTitle') };
}

/**
 * The survey route has no facility segment: `?facility=<code>` picks one of the content facilities,
 * otherwise `content/site.json` `defaultFacilityCode`. The server validates the code again (422).
 */
export default async function SurveyPage({ params, searchParams }: PageProps) {
  const locale = (await params).locale as Locale;
  setRequestLocale(locale);
  const t = await getTranslations({ locale, namespace: 'survey' });
  const requested = (await searchParams).facility;
  const initialFacility =
    typeof requested === 'string' && contentFacilityCodes.includes(requested) ? requested : defaultFacilityCode;
  const facilities = contentFacilityCodes.map((code) => ({
    code,
    name: getFacilityContent(code)?.name[locale] ?? code
  }));

  return (
    <Container size={760} py={{ base: 40, md: 64 }}>
      <Stack gap='lg'>
        <Title order={1} fz={{ base: 26, md: 34 }}>
          {t('title')}
        </Title>
        <Text size='lg' c='dimmed'>
          {t('intro')}
        </Text>
        <InfoBanner tone='info'>
          <Text size='sm'>{t('anonymity')}</Text>
        </InfoBanner>
        <SurveyForm definition={surveyDefinition} facilities={facilities} initialFacility={initialFacility} />
      </Stack>
    </Container>
  );
}

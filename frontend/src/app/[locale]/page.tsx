import { Card, Container, Stack, Text, Title } from '@mantine/core';
import type { Metadata } from 'next';
import { getTranslations, setRequestLocale } from 'next-intl/server';
import { ButtonLink } from '@/components/ButtonLink';
import { InfoBanner } from '@/components/InfoBanner';
import { paths } from '@/config/paths';
import { defaultFacilityCode } from '@/config/site';
import type { Locale } from '@/i18n/routing';

type PageProps = { params: Promise<{ locale: string }> };

export async function generateMetadata({ params }: PageProps): Promise<Metadata> {
  const locale = (await params).locale as Locale;
  const t = await getTranslations({ locale, namespace: 'home' });
  return { title: { absolute: t('metaTitle') } };
}

/** Placeholder home page; the real public content arrives in Phase 5. */
export default async function HomePage({ params }: PageProps) {
  const locale = (await params).locale as Locale;
  setRequestLocale(locale);
  const t = await getTranslations({ locale, namespace: 'home' });

  return (
    <Container size={1200} py={{ base: 40, md: 64 }}>
      <Stack gap='xl'>
        <InfoBanner tone='warning'>{t('placeholder')}</InfoBanner>
        <Stack gap='sm'>
          <Title order={1} fz={{ base: 30, md: 48 }}>
            {t('title')}
          </Title>
          <Text size='lg' c='dimmed'>
            {t('lead')}
          </Text>
        </Stack>
        <Card padding='lg' maw={560}>
          <Stack gap='sm'>
            <Title order={2} fz='xl'>
              {t('registerTitle')}
            </Title>
            <Text>{t('registerBody')}</Text>
            <div>
              <ButtonLink href={paths.registration.wizard.getHref(defaultFacilityCode)}>{t('registerCta')}</ButtonLink>
            </div>
          </Stack>
        </Card>
      </Stack>
    </Container>
  );
}

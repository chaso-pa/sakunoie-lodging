import { Container, Stack } from '@mantine/core';
import type { Metadata } from 'next';
import { setRequestLocale } from 'next-intl/server';
import { AirbnbCta } from '@/components/AirbnbCta';
import { ContentPageHeader, loadPage, MarkdownContent, RatesTable } from '@/features/public-site';
import type { Locale } from '@/i18n/routing';

type PageProps = { params: Promise<{ locale: string }> };

export async function generateMetadata({ params }: PageProps): Promise<Metadata> {
  const { meta } = await loadPage((await params).locale as Locale, 'rates');
  return { title: meta.title, description: meta.description };
}

export default async function RatesPage({ params }: PageProps) {
  const locale = (await params).locale as Locale;
  setRequestLocale(locale);
  const { meta, body } = await loadPage(locale, 'rates');
  return (
    <Container size={760} py={{ base: 40, md: 64 }}>
      <Stack gap='lg'>
        <ContentPageHeader title={meta.title} placeholder={meta.placeholder} />
        <RatesTable rates={meta.rates} disclaimer={meta.disclaimer} />
        <MarkdownContent>{body}</MarkdownContent>
        <div>
          <AirbnbCta size='lg' />
        </div>
      </Stack>
    </Container>
  );
}

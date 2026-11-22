import { Container, Stack } from '@mantine/core';
import type { Metadata } from 'next';
import { setRequestLocale } from 'next-intl/server';
import { ContentPageHeader, loadPage, MarkdownContent } from '@/features/public-site';
import type { Locale } from '@/i18n/routing';

type PageProps = { params: Promise<{ locale: string }> };

export async function generateMetadata({ params }: PageProps): Promise<Metadata> {
  const { meta } = await loadPage((await params).locale as Locale, 'terms');
  return { title: meta.title, description: meta.description };
}

export default async function TermsPage({ params }: PageProps) {
  const locale = (await params).locale as Locale;
  setRequestLocale(locale);
  const { meta, body } = await loadPage(locale, 'terms');
  return (
    <Container size={760} py={{ base: 40, md: 64 }}>
      <Stack gap='lg'>
        <ContentPageHeader
          title={meta.title}
          placeholder={meta.placeholder}
          version={{ version: meta.version, effectiveDate: meta.effectiveDate }}
        />
        <MarkdownContent>{body}</MarkdownContent>
      </Stack>
    </Container>
  );
}

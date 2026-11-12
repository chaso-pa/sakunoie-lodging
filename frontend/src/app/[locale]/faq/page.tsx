import { Container, Stack } from '@mantine/core';
import type { Metadata } from 'next';
import { setRequestLocale } from 'next-intl/server';
import { ContentPageHeader, FaqList, loadPage, MarkdownContent } from '@/features/public-site';
import type { Locale } from '@/i18n/routing';

type PageProps = { params: Promise<{ locale: string }> };

export async function generateMetadata({ params }: PageProps): Promise<Metadata> {
  const { meta } = await loadPage((await params).locale as Locale, 'faq');
  return { title: meta.title, description: meta.description };
}

export default async function FaqPage({ params }: PageProps) {
  const locale = (await params).locale as Locale;
  setRequestLocale(locale);
  const { meta, body } = await loadPage(locale, 'faq');
  return (
    <Container size={760} py={{ base: 40, md: 64 }}>
      <Stack gap='xl'>
        <ContentPageHeader title={meta.title} placeholder={meta.placeholder} />
        {body ? <MarkdownContent>{body}</MarkdownContent> : null}
        <FaqList categories={meta.categories} />
      </Stack>
    </Container>
  );
}

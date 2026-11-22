import { Container, Stack, Text, Title } from '@mantine/core';
import { useTranslations } from 'next-intl';
import { ButtonLink } from '@/components/ButtonLink';
import { paths } from '@/config/paths';

export default function LocaleNotFound() {
  const t = useTranslations('notFound');
  return (
    <Container size={760} py={64}>
      <Stack gap='md'>
        <Title order={1}>{t('title')}</Title>
        <Text>{t('body')}</Text>
        <div>
          <ButtonLink href={paths.public.home.getHref()}>{t('backHome')}</ButtonLink>
        </div>
      </Stack>
    </Container>
  );
}

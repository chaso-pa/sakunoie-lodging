import { Container, Stack, Text } from '@mantine/core';
import { useTranslations } from 'next-intl';

export function SiteFooter() {
  const t = useTranslations('footer');
  return (
    <footer style={{ borderTop: '1px solid var(--sakunoie-border)', background: 'var(--sakunoie-surface)' }}>
      <Container size={1200} py='xl'>
        <Stack gap='xs'>
          <Text size='sm' c='dimmed'>
            {t('tagline')}
          </Text>
          <Text size='sm' c='dimmed'>
            {t('copyright', { year: new Date().getFullYear() })}
          </Text>
        </Stack>
      </Container>
    </footer>
  );
}

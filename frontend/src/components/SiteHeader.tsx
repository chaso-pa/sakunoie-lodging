import { Container, Group, Text } from '@mantine/core';
import { useTranslations } from 'next-intl';
import { LocaleSwitcher } from '@/components/LocaleSwitcher';
import classes from '@/components/SiteHeader.module.scss';
import { paths } from '@/config/paths';
import { Link } from '@/i18n/navigation';

export function SiteHeader() {
  const t = useTranslations();
  return (
    <header className={classes.header}>
      <Container size={1200} h='100%'>
        <Group justify='space-between' h='100%' wrap='nowrap'>
          <Link href={paths.public.home.getHref()} className={classes.brand} aria-label={t('header.homeLabel')}>
            <Text component='span' fw={700} fz='lg' c='forest.6'>
              {t('common.brand')}
            </Text>
            <Text component='span' fz='xs' c='dimmed' className={classes.latin}>
              {t('common.brandLatin')}
            </Text>
          </Link>
          <LocaleSwitcher />
        </Group>
      </Container>
    </header>
  );
}

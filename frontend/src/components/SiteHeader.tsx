import { Container, Group, Text } from '@mantine/core';
import { useTranslations } from 'next-intl';
import { AirbnbCta } from '@/components/AirbnbCta';
import { LocaleSwitcher } from '@/components/LocaleSwitcher';
import classes from '@/components/SiteHeader.module.scss';
import { SiteMobileMenu } from '@/components/SiteMobileMenu';
import { paths } from '@/config/paths';
import { defaultFacility } from '@/config/site';
import { Link } from '@/i18n/navigation';

const navItems = [
  { key: 'facilities', href: paths.public.facilities.getHref() },
  { key: 'guide', href: paths.public.guide.getHref() },
  { key: 'faq', href: paths.public.faq.getHref() },
  { key: 'rates', href: paths.public.rates.getHref() }
] as const;

export function SiteHeader() {
  const t = useTranslations();
  const links = navItems.map((item) => ({ href: item.href, label: t(`nav.${item.key}`) }));
  return (
    <header className={classes.header}>
      <Container size={1200} h='100%'>
        <Group justify='space-between' h='100%' wrap='nowrap' gap='sm'>
          <Link href={paths.public.home.getHref()} className={classes.brand} aria-label={t('header.homeLabel')}>
            <Text component='span' fw={700} fz='lg' c='forest.6'>
              {t('common.brand')}
            </Text>
            <Text component='span' fz='xs' c='dimmed' className={classes.latin}>
              {t('common.brandLatin')}
            </Text>
          </Link>
          <nav aria-label={t('nav.mainLabel')} className={classes.desktopNav}>
            <ul className={classes.navList}>
              {links.map((l) => (
                <li key={l.href}>
                  <Link href={l.href} className={classes.navLink}>
                    {l.label}
                  </Link>
                </li>
              ))}
            </ul>
          </nav>
          <Group gap='sm' wrap='nowrap'>
            <LocaleSwitcher />
            <div className={classes.desktopOnly}>
              <AirbnbCta size='sm' />
            </div>
            <SiteMobileMenu
              links={links}
              labels={{ open: t('nav.openMenu'), title: t('nav.menuTitle'), nav: t('nav.mainLabel') }}
              airbnb={{ href: defaultFacility.airbnbUrl, label: t('cta.airbnb'), newTab: t('cta.opensInNewTab') }}
            />
          </Group>
        </Group>
      </Container>
    </header>
  );
}

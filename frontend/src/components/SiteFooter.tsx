import { Container, SimpleGrid, Stack, Text, Title } from '@mantine/core';
import { useLocale, useTranslations } from 'next-intl';
import classes from '@/components/SiteFooter.module.scss';
import { paths } from '@/config/paths';
import { defaultFacility, defaultFacilityCode } from '@/config/site';
import { Link } from '@/i18n/navigation';
import type { Locale } from '@/i18n/routing';

export function SiteFooter() {
  const t = useTranslations();
  const locale = useLocale() as Locale;
  const groups = [
    {
      title: t('nav.groupStay'),
      links: [
        { href: paths.public.facilities.getHref(), label: t('nav.facilities') },
        { href: paths.public.guide.getHref(), label: t('nav.guide') },
        { href: paths.public.faq.getHref(), label: t('nav.faq') },
        { href: paths.public.rates.getHref(), label: t('nav.rates') }
      ]
    },
    {
      title: t('nav.groupGuests'),
      links: [
        { href: paths.registration.wizard.getHref(defaultFacilityCode), label: t('nav.registration') },
        { href: paths.public.survey.getHref(), label: t('nav.survey') }
      ]
    },
    {
      title: t('nav.groupPolicies'),
      links: [
        { href: paths.public.terms.getHref(), label: t('nav.terms') },
        { href: paths.public.privacy.getHref(), label: t('nav.privacy') }
      ]
    }
  ];
  return (
    <footer className={classes.footer}>
      <Container size={1200} py='xl'>
        <SimpleGrid cols={{ base: 1, xs: 2, md: 4 }} spacing='xl'>
          <Stack gap={6}>
            <Text fw={700} c='forest.6'>
              {defaultFacility.name[locale]}
            </Text>
            <Text size='sm' c='dimmed'>
              {defaultFacility.area[locale]}
            </Text>
            <Text size='sm' c='dimmed'>
              {t('footer.tagline')}
            </Text>
          </Stack>
          {groups.map((g) => (
            <nav key={g.title} aria-label={g.title}>
              <Title order={2} fz='sm' fw={700} mb={4}>
                {g.title}
              </Title>
              <ul className={classes.list}>
                {g.links.map((l) => (
                  <li key={l.href}>
                    <Link href={l.href} className={classes.link}>
                      {l.label}
                    </Link>
                  </li>
                ))}
              </ul>
            </nav>
          ))}
        </SimpleGrid>
        <Text size='sm' c='dimmed' mt='xl'>
          {t('footer.copyright', { year: new Date().getFullYear() })}
        </Text>
      </Container>
    </footer>
  );
}

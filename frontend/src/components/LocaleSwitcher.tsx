'use client';

import { Box, Menu, UnstyledButton } from '@mantine/core';
import { IconCheck, IconChevronDown } from '@tabler/icons-react';
import { CN, JP, KR, US } from 'country-flag-icons/react/3x2';
import { useLocale, useTranslations } from 'next-intl';
import { useState } from 'react';
import classes from '@/components/LocaleSwitcher.module.scss';
import { Link, usePathname } from '@/i18n/navigation';
import { type Locale, routing } from '@/i18n/routing';

// The flag only reinforces the language name, which is always shown next to it.
const flags: Record<Locale, typeof JP> = { ja: JP, en: US, 'zh-CN': CN, ko: KR };

function Flag({ locale }: { locale: Locale }) {
  const Svg = flags[locale];
  return <Svg className={classes.flag} aria-hidden focusable='false' />;
}

export function LocaleSwitcher() {
  const t = useTranslations('localeSwitcher');
  const locale = useLocale() as Locale;
  // Keeps the current path; the URL fragment (edit token) is never part of `pathname`.
  const pathname = usePathname();
  const [opened, setOpened] = useState(false);

  return (
    <Menu
      opened={opened}
      onChange={setOpened}
      position='bottom-end'
      offset={6}
      width={208}
      radius='md'
      shadow='md'
      withinPortal
      transitionProps={{ transition: 'pop-top-right', duration: 120 }}
    >
      <Menu.Target>
        <UnstyledButton
          className={classes.trigger}
          data-locale-switcher
          data-expanded={opened || undefined}
          aria-label={`${t('label')}: ${t(locale)}`}
        >
          <Flag locale={locale} />
          <Box component='span' className={classes.triggerLabel} visibleFrom='xs'>
            {t(locale)}
          </Box>
          <IconChevronDown size={16} className={classes.chevron} aria-hidden />
        </UnstyledButton>
      </Menu.Target>
      <Menu.Dropdown className={classes.dropdown}>
        <Menu.Label>{t('label')}</Menu.Label>
        {routing.locales.map((l) => {
          const active = l === locale;
          return (
            <Menu.Item
              key={l}
              component={Link}
              href={pathname}
              locale={l}
              lang={l}
              hrefLang={l}
              aria-current={active ? 'true' : undefined}
              className={classes.item}
              data-active={active || undefined}
              leftSection={<Flag locale={l} />}
              rightSection={active ? <IconCheck size={16} className={classes.check} aria-hidden /> : null}
            >
              {t(l)}
            </Menu.Item>
          );
        })}
      </Menu.Dropdown>
    </Menu>
  );
}

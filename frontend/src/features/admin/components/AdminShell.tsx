'use client';

import { Box, Burger, Button, Drawer, Group, Stack, Text, UnstyledButton } from '@mantine/core';
import { IconChartBar, IconLogout, IconNotebook } from '@tabler/icons-react';
import Link from 'next/link';
import { usePathname } from 'next/navigation';
import { type ReactNode, useState } from 'react';
import { paths } from '@/config/paths';

const nav = [
  { href: paths.admin.registrations.getHref(), label: '宿泊者名簿', Icon: IconNotebook },
  { href: paths.admin.surveys.getHref(), label: 'アンケート集計', Icon: IconChartBar }
];

const FOREST = 'var(--sakunoie-forest)';
const FOREST_DEEP = '#17382D';

function NavLinks({ onNavigate }: { onNavigate?: () => void }) {
  const pathname = usePathname();
  return (
    <Stack gap={4} component='ul' style={{ listStyle: 'none', margin: 0, padding: 0 }}>
      {nav.map(({ href, label, Icon }) => {
        const active = pathname === href || pathname.startsWith(`${href}/`);
        return (
          <li key={href}>
            <UnstyledButton
              component={Link}
              href={href}
              onClick={onNavigate}
              aria-current={active ? 'page' : undefined}
              px='md'
              py={10}
              style={{
                display: 'flex',
                alignItems: 'center',
                gap: 10,
                minHeight: 44,
                borderRadius: 8,
                color: '#fff',
                fontWeight: active ? 700 : 500,
                background: active ? FOREST_DEEP : 'transparent',
                borderLeft: active ? '4px solid #fff' : '4px solid transparent'
              }}
            >
              <Icon size={20} aria-hidden />
              {label}
            </UnstyledButton>
          </li>
        );
      })}
    </Stack>
  );
}

function Account({ email, onSignOut }: { email: string | null; onSignOut: () => void }) {
  return (
    <Stack gap='xs'>
      {email ? (
        <Text size='sm' c='#fff' style={{ wordBreak: 'break-all' }}>
          {email}
        </Text>
      ) : null}
      <Button
        variant='outline'
        color='gray.0'
        leftSection={<IconLogout size={18} aria-hidden />}
        onClick={onSignOut}
        styles={{ root: { color: '#fff', borderColor: '#fff' } }}
      >
        ログアウト
      </Button>
    </Stack>
  );
}

/** Forest sidebar on desktop, forest top bar with a drawer on mobile (DESIGN §5.4). */
export function AdminShell({
  email,
  onSignOut,
  children
}: {
  email: string | null;
  onSignOut: () => void;
  children: ReactNode;
}) {
  const [opened, setOpened] = useState(false);
  return (
    <Box mih='100dvh' style={{ display: 'flex' }}>
      <Box
        component='nav'
        aria-label='管理メニュー'
        visibleFrom='md'
        w={240}
        p='md'
        bg={FOREST}
        style={{
          flexShrink: 0,
          display: 'flex',
          flexDirection: 'column',
          gap: 24,
          position: 'sticky',
          top: 0,
          height: '100dvh'
        }}
      >
        <Text c='#fff' fw={700} fz='lg'>
          佐久の家 管理
        </Text>
        <Box style={{ flex: 1 }}>
          <NavLinks />
        </Box>
        <Account email={email} onSignOut={onSignOut} />
      </Box>

      <Box style={{ flex: 1, minWidth: 0 }}>
        <Group hiddenFrom='md' h={56} px='md' bg={FOREST} justify='space-between' component='header'>
          <Text c='#fff' fw={700}>
            佐久の家 管理
          </Text>
          <Burger
            opened={opened}
            onClick={() => setOpened((o) => !o)}
            color='#fff'
            aria-label={opened ? 'メニューを閉じる' : 'メニューを開く'}
          />
        </Group>
        <Drawer
          opened={opened}
          onClose={() => setOpened(false)}
          title='メニュー'
          hiddenFrom='md'
          size={280}
          styles={{ content: { background: FOREST }, header: { background: FOREST, color: '#fff' } }}
        >
          <Stack gap='xl'>
            <NavLinks onNavigate={() => setOpened(false)} />
            <Account email={email} onSignOut={onSignOut} />
          </Stack>
        </Drawer>
        <Box component='main' id='main' p={{ base: 'md', md: 'xl' }} maw={1400}>
          {children}
        </Box>
      </Box>
    </Box>
  );
}

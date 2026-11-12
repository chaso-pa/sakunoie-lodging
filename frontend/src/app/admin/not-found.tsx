'use client';

import { Anchor, Center, Stack, Text, Title } from '@mantine/core';
import Link from 'next/link';
import { paths } from '@/config/paths';

export default function AdminNotFound() {
  return (
    <Center mih='100dvh' p='md'>
      <Stack component='main' id='main' maw={480} gap='sm'>
        <Title order={1} fz={26}>
          ページが見つかりません
        </Title>
        <Text>URL を確認するか、宿泊者名簿から目的の登録を開いてください。</Text>
        <Anchor component={Link} href={paths.admin.registrations.getHref()} fw={600}>
          宿泊者名簿へ
        </Anchor>
      </Stack>
    </Center>
  );
}

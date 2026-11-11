'use client';

import { Anchor, Card, Group, Stack, Table, Text, VisuallyHidden } from '@mantine/core';
import Link from 'next/link';
import { paths } from '@/config/paths';
import { completenessChip, identityChip } from '@/features/admin/components/StatusChip';
import type { AdminRegistrationRow } from '@/features/admin/types/api';
import { formatDate } from '@/features/admin/utils/format';

const stay = (r: AdminRegistrationRow) => `${formatDate(r.check_in_date)} – ${formatDate(r.check_out_date)}`;

/** Ledger as a table on desktop and stacked cards on mobile (DESIGN §5.4). */
export function LedgerTable({ rows, showFacility }: { rows: AdminRegistrationRow[]; showFacility: boolean }) {
  return (
    <>
      <Table.ScrollContainer minWidth={900} visibleFrom='md'>
        <Table striped highlightOnHover verticalSpacing='sm' bg='var(--sakunoie-surface)'>
          <Table.Thead>
            <Table.Tr>
              <Table.Th>宿泊日</Table.Th>
              {showFacility ? <Table.Th>施設</Table.Th> : null}
              <Table.Th>予約番号</Table.Th>
              <Table.Th>代表者</Table.Th>
              <Table.Th>予定人数</Table.Th>
              <Table.Th>登録人数</Table.Th>
              <Table.Th>本人確認</Table.Th>
              <Table.Th>登録状況</Table.Th>
              <Table.Th>
                <VisuallyHidden>操作</VisuallyHidden>
              </Table.Th>
            </Table.Tr>
          </Table.Thead>
          <Table.Tbody>
            {rows.map((r) => (
              <Table.Tr key={r.public_id}>
                <Table.Td style={{ whiteSpace: 'nowrap' }}>{stay(r)}</Table.Td>
                {showFacility ? <Table.Td>{r.facility.name}</Table.Td> : null}
                <Table.Td>{r.booking_reference ?? '—'}</Table.Td>
                <Table.Td>{r.representative_name}</Table.Td>
                <Table.Td>{r.planned_guest_count}名</Table.Td>
                <Table.Td>{r.registered_count}名</Table.Td>
                <Table.Td>{identityChip(r.identity_verified)}</Table.Td>
                <Table.Td>{completenessChip(r.registered_count, r.planned_guest_count, r.remaining_count)}</Table.Td>
                <Table.Td>
                  <Anchor component={Link} href={paths.admin.registration.getHref(r.public_id)} fw={600}>
                    詳細<VisuallyHidden>（{r.representative_name}）</VisuallyHidden>
                  </Anchor>
                </Table.Td>
              </Table.Tr>
            ))}
          </Table.Tbody>
        </Table>
      </Table.ScrollContainer>

      <Stack hiddenFrom='md' gap='sm' component='ul' style={{ listStyle: 'none', padding: 0, margin: 0 }}>
        {rows.map((r) => (
          <Card key={r.public_id} component='li' padding='md'>
            <Stack gap={6}>
              <Group justify='space-between' wrap='nowrap'>
                <Text fw={700}>{stay(r)}</Text>
                <Anchor component={Link} href={paths.admin.registration.getHref(r.public_id)} fw={600}>
                  詳細<VisuallyHidden>（{r.representative_name}）</VisuallyHidden>
                </Anchor>
              </Group>
              <Text>{r.representative_name}</Text>
              <Text size='sm' c='dimmed'>
                {showFacility ? `${r.facility.name} ・ ` : ''}予約番号 {r.booking_reference ?? '—'} ・ 予定{' '}
                {r.planned_guest_count}名 / 登録 {r.registered_count}名
              </Text>
              <Group gap='xs'>
                {identityChip(r.identity_verified)}
                {completenessChip(r.registered_count, r.planned_guest_count, r.remaining_count)}
              </Group>
            </Stack>
          </Card>
        ))}
      </Stack>
    </>
  );
}

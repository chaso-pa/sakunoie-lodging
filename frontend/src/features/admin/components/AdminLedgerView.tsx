'use client';

import { Group, Loader, Pagination, SimpleGrid, Stack, Text, Title } from '@mantine/core';
import { IconAlertTriangle, IconNotebook, IconShieldCheck, IconUsers } from '@tabler/icons-react';
import { usePathname, useRouter, useSearchParams } from 'next/navigation';
import { useCallback, useMemo } from 'react';
import { InfoBanner } from '@/components/InfoBanner';
import { useAdminRegistrations } from '@/features/admin/api/get-registrations';
import { ExportAction } from '@/features/admin/components/ExportAction';
import { FacilityFilter } from '@/features/admin/components/FacilityFilter';
import { LedgerTable } from '@/features/admin/components/LedgerTable';
import { MetricCard } from '@/features/admin/components/MetricCard';
import { useAdminSession } from '@/features/admin/hooks/use-admin-session';
import type { LedgerFilter } from '@/features/admin/types/api';
import { adminErrorMessage } from '@/features/admin/utils/error-message';

const DATE_RE = /^\d{4}-\d{2}-\d{2}$/;

/** Filter state lives in the URL so back navigation and reloads keep it. */
const filterFromParams = (sp: URLSearchParams): LedgerFilter => {
  const facilityId = Number(sp.get('facility_id'));
  const page = Number(sp.get('page'));
  const from = sp.get('from') ?? '';
  const to = sp.get('to') ?? '';
  return {
    facilityId: Number.isInteger(facilityId) && facilityId > 0 ? facilityId : undefined,
    from: DATE_RE.test(from) ? from : undefined,
    to: DATE_RE.test(to) ? to : undefined,
    q: sp.get('q')?.slice(0, 100) || undefined,
    page: Number.isInteger(page) && page > 0 ? page : 1
  };
};

const paramsFromFilter = (f: LedgerFilter) => {
  const sp = new URLSearchParams();
  if (f.facilityId) sp.set('facility_id', String(f.facilityId));
  if (f.from) sp.set('from', f.from);
  if (f.to) sp.set('to', f.to);
  if (f.q) sp.set('q', f.q);
  if (f.page && f.page > 1) sp.set('page', String(f.page));
  return sp.toString();
};

export function AdminLedgerView() {
  const { uid, facilities } = useAdminSession();
  const router = useRouter();
  const pathname = usePathname();
  const searchParams = useSearchParams();
  const filter = useMemo(() => filterFromParams(new URLSearchParams(searchParams.toString())), [searchParams]);
  const setFilter = useCallback(
    (f: LedgerFilter) => {
      const qs = paramsFromFilter(f);
      router.replace(qs ? `${pathname}?${qs}` : pathname);
    },
    [router, pathname]
  );
  const query = useAdminRegistrations(uid, filter);
  const exportFacility = filter.facilityId ?? (facilities.length === 1 ? facilities[0].id : undefined);
  const data = query.data;

  return (
    <Stack gap='lg'>
      <Group justify='space-between' align='flex-end'>
        <Title order={1} fz={{ base: 24, md: 30 }}>
          宿泊者名簿
        </Title>
        <ExportAction facilityId={exportFacility} from={filter.from} to={filter.to} />
      </Group>

      <FacilityFilter facilities={facilities} value={filter} onChange={setFilter} />

      {query.isError ? (
        <InfoBanner tone='error' announce='assertive'>
          {adminErrorMessage(query.error)}
        </InfoBanner>
      ) : null}

      {data ? (
        <SimpleGrid cols={{ base: 2, md: 4 }} spacing='sm'>
          <MetricCard
            label='登録件数'
            value={data.metrics.registrations}
            icon={<IconNotebook size={16} aria-hidden />}
          />
          <MetricCard label='登録済み人数' value={data.metrics.guests} icon={<IconUsers size={16} aria-hidden />} />
          <MetricCard
            label='未登録あり'
            value={data.metrics.incomplete}
            icon={<IconAlertTriangle size={16} aria-hidden />}
          />
          <MetricCard
            label='本人確認済み'
            value={data.metrics.verified}
            icon={<IconShieldCheck size={16} aria-hidden />}
          />
        </SimpleGrid>
      ) : null}

      {query.isPending ? (
        <Group justify='center' p='xl'>
          <Loader color='forest' aria-label='読み込み中' />
        </Group>
      ) : null}

      {data && data.items.length === 0 ? (
        <InfoBanner
          title={filter.q || filter.from || filter.to ? '条件に一致する登録はありません' : '登録はまだありません'}
        >
          {filter.q || filter.from || filter.to ? '検索語や期間を変えてお試しください。' : null}
        </InfoBanner>
      ) : null}

      {data && data.items.length > 0 ? (
        <Stack gap='sm'>
          <Text size='sm' c='dimmed' aria-live='polite'>
            {data.total}件中 {(data.page - 1) * data.page_size + 1}–
            {(data.page - 1) * data.page_size + data.items.length}
            件を表示
          </Text>
          <LedgerTable rows={data.items} showFacility={facilities.length > 1} />
          {data.total > data.page_size ? (
            <Pagination
              total={Math.ceil(data.total / data.page_size)}
              value={data.page}
              onChange={(page) => setFilter({ ...filter, page })}
              color='forest'
              getControlProps={(control) => ({
                'aria-label': control === 'next' ? '次のページ' : control === 'previous' ? '前のページ' : control
              })}
            />
          ) : null}
        </Stack>
      ) : null}
    </Stack>
  );
}

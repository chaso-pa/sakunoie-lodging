'use client';

import { Button, Stack, Text } from '@mantine/core';
import { IconFileDownload } from '@tabler/icons-react';
import { InfoBanner } from '@/components/InfoBanner';
import { useExportRegistrations } from '@/features/admin/api/export-registrations';
import { adminErrorMessage } from '@/features/admin/utils/error-message';

type Props = { facilityId?: number; from?: string; to?: string };

/** CSV export with preparing/success/failure states. The server audits every export. */
export function ExportAction({ facilityId, from, to }: Props) {
  const mutation = useExportRegistrations();
  return (
    <Stack gap='xs' align='flex-start'>
      <Button
        variant='outline'
        color='forest'
        leftSection={<IconFileDownload size={18} aria-hidden />}
        disabled={!facilityId}
        loading={mutation.isPending}
        onClick={() => facilityId && mutation.mutate({ facilityId, from, to })}
      >
        CSV出力
      </Button>
      {!facilityId ? (
        <Text size='sm' c='dimmed'>
          CSV出力は施設を1つ選んでから行えます。
        </Text>
      ) : null}
      <div aria-live='polite'>
        {mutation.isPending ? <Text size='sm'>CSVを準備しています…</Text> : null}
        {mutation.isSuccess ? (
          <InfoBanner tone='success'>
            {mutation.data} をダウンロードしました。出力は監査ログに記録されています。
          </InfoBanner>
        ) : null}
        {mutation.isError ? (
          <InfoBanner tone='error'>CSVを出力できませんでした。{adminErrorMessage(mutation.error)}</InfoBanner>
        ) : null}
      </div>
    </Stack>
  );
}

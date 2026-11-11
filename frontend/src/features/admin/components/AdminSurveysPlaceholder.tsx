import { Stack, Text, Title } from '@mantine/core';
import { InfoBanner } from '@/components/InfoBanner';

/** Phase 5 replaces this with the aggregate-only survey summary. */
export function AdminSurveysPlaceholder() {
  return (
    <Stack gap='lg'>
      <Title order={1} fz={{ base: 24, md: 30 }}>
        アンケート集計
      </Title>
      <InfoBanner title='準備中'>
        <Text>匿名アンケートの集計画面は次の段階で追加します。個別の回答は表示せず、集計値のみを表示します。</Text>
      </InfoBanner>
    </Stack>
  );
}

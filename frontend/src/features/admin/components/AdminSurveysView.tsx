'use client';

import { Button, Group, Loader, NativeSelect, SimpleGrid, Stack, Text, TextInput, Title } from '@mantine/core';
import { IconFilter, IconMessageCircle } from '@tabler/icons-react';
import { type FormEvent, useState } from 'react';
import { InfoBanner } from '@/components/InfoBanner';
import { surveyDefinition } from '@/config/survey-definition';
import { useSurveySummary } from '@/features/admin/api/get-survey-summary';
import { MetricCard } from '@/features/admin/components/MetricCard';
import { SurveySummary } from '@/features/admin/components/SurveySummary';
import { useAdminSession } from '@/features/admin/hooks/use-admin-session';
import type { SurveySummaryFilter } from '@/features/admin/types/api';
import { adminErrorMessage } from '@/features/admin/utils/error-message';

/** Anonymous survey aggregates (contract §3.7): counts and means only, small cells suppressed. */
export function AdminSurveysView() {
  const { uid, facilities } = useAdminSession();
  const [filter, setFilter] = useState<SurveySummaryFilter>({});
  const [draft, setDraft] = useState<SurveySummaryFilter>({});
  const query = useSurveySummary(uid, filter);
  const data = query.data;

  const submit = (e: FormEvent) => {
    e.preventDefault();
    setFilter(draft);
  };

  return (
    <Stack gap='lg'>
      <Title order={1} fz={{ base: 24, md: 30 }}>
        アンケート集計
      </Title>
      <Text c='dimmed'>
        匿名アンケートの集計値のみを表示します。個別の回答や自由記述の内容は表示しません。日付は回答日（UTC）で絞り込みます。
      </Text>
      <form onSubmit={submit} aria-label='絞り込み'>
        <Group align='flex-end' gap='sm' wrap='wrap'>
          {facilities.length > 1 ? (
            <NativeSelect
              label='施設'
              value={draft.facilityId ? String(draft.facilityId) : ''}
              onChange={(e) =>
                setDraft({ ...draft, facilityId: e.currentTarget.value ? Number(e.currentTarget.value) : undefined })
              }
              data={[
                { value: '', label: 'すべての施設' },
                ...facilities.map((f) => ({ value: String(f.id), label: f.name }))
              ]}
              w={{ base: '100%', sm: 220 }}
            />
          ) : null}
          <TextInput
            label='回答日（から）'
            type='date'
            value={draft.from ?? ''}
            onChange={(e) => setDraft({ ...draft, from: e.currentTarget.value || undefined })}
            w={{ base: '100%', sm: 180 }}
          />
          <TextInput
            label='回答日（まで）'
            type='date'
            value={draft.to ?? ''}
            onChange={(e) => setDraft({ ...draft, to: e.currentTarget.value || undefined })}
            w={{ base: '100%', sm: 180 }}
          />
          <Button type='submit' color='forest' leftSection={<IconFilter size={18} aria-hidden />}>
            絞り込む
          </Button>
        </Group>
      </form>

      {query.isError ? (
        <InfoBanner tone='error' announce='assertive'>
          <Text size='sm'>{adminErrorMessage(query.error)}</Text>
        </InfoBanner>
      ) : null}
      {query.isPending ? <Loader color='forest' aria-label='読み込み中' /> : null}

      {data ? (
        <>
          <SimpleGrid cols={{ base: 2, md: 3 }} spacing='md'>
            <MetricCard label='回答数' value={data.total} icon={<IconMessageCircle size={16} aria-hidden />} />
            <MetricCard label='表示の下限' value={`${data.min_responses}件`} />
          </SimpleGrid>
          {data.total === 0 ? (
            <InfoBanner title='回答がありません'>
              <Text size='sm'>条件に合う回答はまだありません。</Text>
            </InfoBanner>
          ) : data.total < data.min_responses ? (
            <InfoBanner tone='warning' title='回答数が少ないため集計値を表示していません'>
              <Text size='sm'>
                回答者の特定を防ぐため、回答が{data.min_responses}件未満の設問は平均や内訳を表示しません。
              </Text>
            </InfoBanner>
          ) : null}
          <SimpleGrid cols={{ base: 1, lg: 2 }} spacing='md'>
            {data.questions.map((q, i) => (
              <SurveySummary
                key={q.code}
                summary={q}
                index={i + 1}
                minResponses={data.min_responses}
                question={surveyDefinition.questions.find((d) => d.code === q.code)}
              />
            ))}
          </SimpleGrid>
        </>
      ) : null}
    </Stack>
  );
}

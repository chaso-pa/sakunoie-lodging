'use client';

import {
  Button,
  Card,
  Group,
  Loader,
  NativeSelect,
  Pagination,
  SimpleGrid,
  Stack,
  Table,
  Text,
  TextInput,
  Title,
  useMatches
} from '@mantine/core';
import { DatePickerInput, DatesProvider } from '@mantine/dates';
import { IconBan, IconCalendar, IconLinkPlus } from '@tabler/icons-react';
import dayjs from 'dayjs';
import 'dayjs/locale/ja';
import { type FormEvent, useState } from 'react';
import { InfoBanner } from '@/components/InfoBanner';
import { useAdminInvitations, useCreateInvitation, useRevokeInvitation } from '@/features/admin/api/invitations';
import { ConfirmDialog } from '@/features/admin/components/ConfirmDialog';
import { InvitationCreatedDialog } from '@/features/admin/components/InvitationCreatedDialog';
import { invitationChip, invitationStatusLabel } from '@/features/admin/components/StatusChip';
import { useAdminSession } from '@/features/admin/hooks/use-admin-session';
import type {
  AdminInvitation,
  CreateInvitationResult,
  InvitationFilter,
  InvitationStatus
} from '@/features/admin/types/api';
import { adminErrorMessage } from '@/features/admin/utils/error-message';
import { formatDate, formatInstant } from '@/features/admin/utils/format';

type Draft = { facilityId: string; checkIn: string; checkOut: string; bookingReference: string; note: string };

const STATUSES: InvitationStatus[] = ['active', 'used', 'expired', 'revoked'];

/** Nights between two `YYYY-MM-DD` dates (UTC arithmetic, so no DST or time-zone drift). */
const nightsBetween = (checkIn: string, checkOut: string) =>
  Math.round((Date.parse(`${checkOut}T00:00:00Z`) - Date.parse(`${checkIn}T00:00:00Z`)) / 86_400_000);

const stay = (i: AdminInvitation) => `${formatDate(i.check_in_date)} – ${formatDate(i.check_out_date)}`;

/**
 * Per-booking invitation links (root CLAUDE.md §13, 2026-11-15): the host creates one per booking and
 * sends it in the Airbnb message. A guest can register only with a valid invitation, once.
 */
export function AdminInvitationsView() {
  const { uid, facilities } = useAdminSession();
  const [draft, setDraft] = useState<Draft>({
    facilityId: facilities[0] ? String(facilities[0].id) : '',
    checkIn: '',
    checkOut: '',
    bookingReference: '',
    note: ''
  });
  const [formError, setFormError] = useState<string | null>(null);
  const [created, setCreated] = useState<CreateInvitationResult | null>(null);
  const [filter, setFilter] = useState<InvitationFilter>({});
  const [revoking, setRevoking] = useState<AdminInvitation | null>(null);
  const [notice, setNotice] = useState<string | null>(null);
  const create = useCreateInvitation(uid);
  const revoke = useRevokeInvitation(uid);
  const list = useAdminInvitations(uid, filter);
  // Two months side by side like Airbnb's date picker; one month in a full-screen modal on phones.
  const wide = useMatches({ base: false, sm: true });
  const nights = draft.checkIn && draft.checkOut ? nightsBetween(draft.checkIn, draft.checkOut) : 0;
  const timezoneOf = (facilityId: number) => facilities.find((f) => f.id === facilityId)?.timezone ?? 'Asia/Tokyo';

  const submit = (e: FormEvent) => {
    e.preventDefault();
    setNotice(null);
    if (!draft.facilityId || !draft.checkIn || !draft.checkOut) {
      setFormError('施設と、チェックイン・チェックアウトの日付を選んでください。');
      return;
    }
    if (draft.checkOut <= draft.checkIn) {
      setFormError('チェックアウト日はチェックイン日より後にしてください。');
      return;
    }
    setFormError(null);
    create.mutate(
      {
        facility_id: Number(draft.facilityId),
        check_in_date: draft.checkIn,
        check_out_date: draft.checkOut,
        ...(draft.bookingReference.trim() ? { booking_reference: draft.bookingReference.trim() } : {}),
        ...(draft.note.trim() ? { note: draft.note.trim() } : {})
      },
      {
        onSuccess: (result) => {
          setCreated(result);
          setDraft((d) => ({ ...d, checkIn: '', checkOut: '', bookingReference: '', note: '' }));
        }
      }
    );
  };

  const confirmRevoke = () => {
    if (!revoking) return;
    revoke.mutate(revoking.public_id, {
      onSuccess: () => {
        setNotice('招待リンクを取り消しました。このリンクではもう登録できません。');
        setRevoking(null);
      }
    });
  };

  const revokeButton = (i: AdminInvitation) =>
    i.status === 'active' ? (
      <Button
        size='xs'
        variant='outline'
        color='red'
        leftSection={<IconBan size={16} aria-hidden />}
        onClick={() => {
          revoke.reset();
          setRevoking(i);
        }}
        aria-label={`取り消す（${stay(i)}${i.note ? `、${i.note}` : ''}）`}
      >
        取り消す
      </Button>
    ) : null;

  const data = list.data;
  const showFacility = facilities.length > 1;

  return (
    <Stack gap='lg'>
      <Title order={1} fz={{ base: 24, md: 30 }}>
        招待リンク
      </Title>
      <Text c='dimmed'>
        宿泊者登録は、予約ごとに作成する招待リンクからだけ行えます。作成したリンクを Airbnb
        のメッセージでゲストに送ってください。リンクはチェックアウト日の終わり（施設の現地時刻）まで有効で、1回だけ使えます。
      </Text>

      <Card padding='lg' component='section' aria-labelledby='invite-create-title'>
        <Stack gap='md'>
          <Title order={2} fz='lg' id='invite-create-title'>
            招待リンクを作成
          </Title>
          <form onSubmit={submit} noValidate aria-labelledby='invite-create-title'>
            <Stack gap='md'>
              <SimpleGrid cols={{ base: 1, sm: 2 }} spacing='md'>
                {showFacility ? (
                  <NativeSelect
                    label='施設'
                    withAsterisk
                    value={draft.facilityId}
                    onChange={(e) => setDraft({ ...draft, facilityId: e.currentTarget.value })}
                    data={facilities.map((f) => ({ value: String(f.id), label: f.name }))}
                  />
                ) : null}
                <DatesProvider settings={{ locale: 'ja', firstDayOfWeek: 0 }}>
                  <DatePickerInput
                    type='range'
                    label='チェックイン・チェックアウト'
                    description={
                      nights > 0 ? `${nights}泊` : 'カレンダーでチェックイン日とチェックアウト日を選びます。'
                    }
                    placeholder='日付を選択'
                    withAsterisk
                    clearable
                    leftSection={<IconCalendar size={18} aria-hidden />}
                    valueFormat='YYYY年M月D日(dd)'
                    labelSeparator='〜'
                    numberOfColumns={wide ? 2 : 1}
                    dropdownType={wide ? 'popover' : 'modal'}
                    modalProps={{ centered: true }}
                    monthLabelFormat='YYYY年M月'
                    // Outside days would repeat the next column's dates in the two-month view.
                    hideOutsideDates
                    getDayAriaLabel={(date) => dayjs(date).format('YYYY年M月D日')}
                    ariaLabels={{
                      previousMonth: '前の月',
                      nextMonth: '次の月',
                      previousYear: '前の年',
                      nextYear: '次の年',
                      previousDecade: '前の10年',
                      nextDecade: '次の10年',
                      monthLevelControl: '月を選ぶ',
                      yearLevelControl: '年を選ぶ'
                    }}
                    value={[draft.checkIn || null, draft.checkOut || null]}
                    onChange={([checkIn, checkOut]) =>
                      setDraft({ ...draft, checkIn: checkIn ?? '', checkOut: checkOut ?? '' })
                    }
                  />
                </DatesProvider>
                <TextInput
                  label='予約コード（任意）'
                  description='Airbnb の予約コードなど。ゲストには表示されません。'
                  maxLength={255}
                  autoComplete='off'
                  value={draft.bookingReference}
                  onChange={(e) => setDraft({ ...draft, bookingReference: e.currentTarget.value })}
                />
                <TextInput
                  label='メモ（任意）'
                  description='ゲスト名など、ホスト用のメモです。ゲストには表示されません。'
                  maxLength={255}
                  autoComplete='off'
                  value={draft.note}
                  onChange={(e) => setDraft({ ...draft, note: e.currentTarget.value })}
                />
              </SimpleGrid>
              {formError ? (
                <InfoBanner tone='error' announce='assertive'>
                  <Text size='sm'>{formError}</Text>
                </InfoBanner>
              ) : null}
              {create.isError ? (
                <InfoBanner tone='error' announce='assertive'>
                  <Text size='sm'>{adminErrorMessage(create.error)}</Text>
                </InfoBanner>
              ) : null}
              <div>
                <Button
                  type='submit'
                  color='forest'
                  loading={create.isPending}
                  leftSection={<IconLinkPlus size={18} aria-hidden />}
                >
                  招待リンクを作成
                </Button>
              </div>
            </Stack>
          </form>
        </Stack>
      </Card>

      <Stack gap='sm' component='section' aria-labelledby='invite-list-title'>
        <Group justify='space-between' align='flex-end' wrap='wrap'>
          <Title order={2} fz='lg' id='invite-list-title'>
            作成した招待リンク
          </Title>
          <NativeSelect
            label='状態'
            value={filter.status ?? ''}
            onChange={(e) =>
              setFilter({
                ...filter,
                status: (e.currentTarget.value || undefined) as InvitationStatus | undefined,
                page: 1
              })
            }
            data={[
              { value: '', label: 'すべて' },
              ...STATUSES.map((s) => ({ value: s, label: invitationStatusLabel[s] }))
            ]}
            w={{ base: '100%', sm: 200 }}
          />
        </Group>
        {notice ? (
          <InfoBanner tone='success' announce='polite'>
            <Text size='sm'>{notice}</Text>
          </InfoBanner>
        ) : null}
        {list.isError ? (
          <InfoBanner tone='error' announce='assertive'>
            <Text size='sm'>{adminErrorMessage(list.error)}</Text>
          </InfoBanner>
        ) : null}
        {list.isPending ? <Loader color='forest' aria-label='読み込み中' /> : null}
        {data && data.items.length === 0 ? <Text c='dimmed'>条件に合う招待リンクはありません。</Text> : null}
        {data && data.items.length > 0 ? (
          <>
            <Table.ScrollContainer minWidth={900} visibleFrom='md'>
              <Table striped verticalSpacing='sm' bg='var(--sakunoie-surface)'>
                <Table.Thead>
                  <Table.Tr>
                    <Table.Th>宿泊日</Table.Th>
                    {showFacility ? <Table.Th>施設</Table.Th> : null}
                    <Table.Th>予約コード</Table.Th>
                    <Table.Th>メモ</Table.Th>
                    <Table.Th>状態</Table.Th>
                    <Table.Th>有効期限</Table.Th>
                    <Table.Th>操作</Table.Th>
                  </Table.Tr>
                </Table.Thead>
                <Table.Tbody>
                  {data.items.map((i) => (
                    <Table.Tr key={i.public_id}>
                      <Table.Td style={{ whiteSpace: 'nowrap' }}>{stay(i)}</Table.Td>
                      {showFacility ? <Table.Td>{i.facility.name}</Table.Td> : null}
                      <Table.Td>{i.booking_reference ?? '—'}</Table.Td>
                      <Table.Td>{i.note ?? '—'}</Table.Td>
                      <Table.Td>{invitationChip(i.status)}</Table.Td>
                      <Table.Td style={{ whiteSpace: 'nowrap' }}>
                        {formatInstant(i.expires_at, timezoneOf(i.facility.id))}
                      </Table.Td>
                      <Table.Td>{revokeButton(i)}</Table.Td>
                    </Table.Tr>
                  ))}
                </Table.Tbody>
              </Table>
            </Table.ScrollContainer>
            <Stack hiddenFrom='md' gap='sm' component='ul' style={{ listStyle: 'none', padding: 0, margin: 0 }}>
              {data.items.map((i) => (
                <Card key={i.public_id} component='li' padding='md'>
                  <Stack gap={6}>
                    <Group justify='space-between' wrap='nowrap'>
                      <Text fw={700}>{stay(i)}</Text>
                      {invitationChip(i.status)}
                    </Group>
                    <Text size='sm' c='dimmed'>
                      {showFacility ? `${i.facility.name} ・ ` : ''}予約コード {i.booking_reference ?? '—'} ・ メモ{' '}
                      {i.note ?? '—'}
                    </Text>
                    <Text size='sm' c='dimmed'>
                      有効期限 {formatInstant(i.expires_at, timezoneOf(i.facility.id))}
                    </Text>
                    {revokeButton(i)}
                  </Stack>
                </Card>
              ))}
            </Stack>
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
          </>
        ) : null}
      </Stack>

      <InvitationCreatedDialog
        result={created}
        timezone={created ? timezoneOf(created.invitation.facility.id) : 'Asia/Tokyo'}
        onClose={() => setCreated(null)}
      />
      <ConfirmDialog
        opened={revoking !== null}
        title='招待リンクを取り消しますか'
        confirmLabel='取り消す'
        danger
        loading={revoke.isPending}
        onConfirm={confirmRevoke}
        onClose={() => setRevoking(null)}
      >
        <Stack gap='xs'>
          <Text size='sm'>
            {revoking ? `${stay(revoking)}${revoking.note ? `（${revoking.note}）` : ''}` : ''}{' '}
            の招待リンクを取り消します。このリンクではもう登録できなくなります。必要なら新しい招待リンクを作成してください。
          </Text>
          {revoke.isError ? (
            <InfoBanner tone='error' announce='assertive'>
              <Text size='sm'>{adminErrorMessage(revoke.error)}</Text>
            </InfoBanner>
          ) : null}
        </Stack>
      </ConfirmDialog>
    </Stack>
  );
}

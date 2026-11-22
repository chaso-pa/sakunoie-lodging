'use client';

import { Anchor, Badge, Button, Card, Group, Loader, SimpleGrid, Stack, Text, Title } from '@mantine/core';
import { IconArrowLeft, IconEye, IconEyeOff, IconLink, IconPencil, IconShieldCheck } from '@tabler/icons-react';
import Link from 'next/link';
import { type ReactNode, useState } from 'react';
import { InfoBanner } from '@/components/InfoBanner';
import { paths } from '@/config/paths';
import { useAdminRegistration } from '@/features/admin/api/get-registration';
import { useReissueEditLink, useVerifyIdentity } from '@/features/admin/api/registration-actions';
import { ConfirmDialog } from '@/features/admin/components/ConfirmDialog';
import { CorrectionForm } from '@/features/admin/components/CorrectionForm';
import { ReissueLinkDialog } from '@/features/admin/components/ReissueLinkDialog';
import { completenessChip, identityChip } from '@/features/admin/components/StatusChip';
import { useAdminSession } from '@/features/admin/hooks/use-admin-session';
import type { AdminGuest, ReissueEditLinkResult } from '@/features/admin/types/api';
import { adminErrorMessage } from '@/features/admin/utils/error-message';
import { formatDate, formatInstant } from '@/features/admin/utils/format';

function Field({ label, children }: { label: string; children: ReactNode }) {
  return (
    <div>
      <Text size='sm' c='dimmed' fw={600}>
        {label}
      </Text>
      <Text style={{ wordBreak: 'break-word' }}>{children}</Text>
    </div>
  );
}

function GuestCard({ guest, index }: { guest: AdminGuest; index: number }) {
  return (
    <Card padding='md' component='li'>
      <Stack gap='sm'>
        <Group gap='xs'>
          <Text fw={700}>
            宿泊者 {index + 1}: {guest.full_name}
          </Text>
          {guest.is_representative ? (
            <Badge color='forest' variant='filled'>
              代表者
            </Badge>
          ) : null}
        </Group>
        <SimpleGrid cols={{ base: 1, sm: 2, lg: 3 }} spacing='sm'>
          <Field label='フリガナ'>{guest.name_kana ?? '—'}</Field>
          <Field label='住所'>{guest.address}</Field>
          <Field label='電話番号'>{guest.phone ?? '—'}</Field>
          <Field label='メールアドレス'>{guest.email ?? '—'}</Field>
          <Field label='日本国内の住所'>{guest.has_japan_address ? 'あり' : 'なし'}</Field>
          {!guest.has_japan_address ? (
            <>
              <Field label='国籍'>{guest.nationality ?? '—'}</Field>
              <Field label='旅券番号（下4桁）'>{guest.passport_last4 ? `****${guest.passport_last4}` : '未登録'}</Field>
            </>
          ) : null}
        </SimpleGrid>
      </Stack>
    </Card>
  );
}

type Dialog = 'reveal' | 'verify' | 'reissue' | 'correct' | null;

export function AdminRegistrationDetailView({ id }: { id: string }) {
  const { uid } = useAdminSession();
  const [reveal, setReveal] = useState(false);
  const [dialog, setDialog] = useState<Dialog>(null);
  const [link, setLink] = useState<ReissueEditLinkResult | null>(null);
  const query = useAdminRegistration(uid, id, reveal);
  const verify = useVerifyIdentity(uid, id);
  const reissue = useReissueEditLink(uid, id);
  const d = query.data;

  const back = (
    <Anchor
      component={Link}
      href={paths.admin.registrations.getHref()}
      style={{ display: 'inline-flex', alignItems: 'center', gap: 4 }}
    >
      <IconArrowLeft size={18} aria-hidden />
      宿泊者名簿に戻る
    </Anchor>
  );

  if (query.isPending) {
    return (
      <Stack>
        {back}
        <Loader color='forest' aria-label='読み込み中' />
      </Stack>
    );
  }
  if (!d) {
    return (
      <Stack>
        {back}
        <InfoBanner tone='error' announce='assertive'>
          {adminErrorMessage(query.error)}
        </InfoBanner>
      </Stack>
    );
  }
  const tz = d.facility.timezone;
  const actionError = verify.error ?? reissue.error;

  return (
    <Stack gap='lg'>
      {back}
      <Stack gap='xs'>
        <Title order={1} fz={{ base: 22, md: 28 }}>
          {formatDate(d.check_in_date)} – {formatDate(d.check_out_date)} の宿泊
        </Title>
        <Group gap='xs'>
          <Text c='dimmed'>{d.facility.name}</Text>
          {identityChip(d.identity_verified)}
          {completenessChip(d.registered_count, d.planned_guest_count, d.remaining_count)}
        </Group>
      </Stack>

      {actionError ? (
        <InfoBanner tone='error' announce='assertive'>
          {adminErrorMessage(actionError)}
        </InfoBanner>
      ) : null}
      {verify.isSuccess ? (
        <InfoBanner tone='success' announce='polite'>
          本人確認を記録しました。
        </InfoBanner>
      ) : null}

      <Group gap='sm'>
        <Button
          color='forest'
          leftSection={<IconShieldCheck size={18} aria-hidden />}
          onClick={() => setDialog('verify')}
          disabled={d.identity_verified}
        >
          {d.identity_verified ? '本人確認済み' : '本人確認済みにする'}
        </Button>
        <Button
          variant='outline'
          color='forest'
          leftSection={<IconPencil size={18} aria-hidden />}
          onClick={() => setDialog('correct')}
        >
          訂正する
        </Button>
        <Button
          variant='outline'
          color='forest'
          leftSection={<IconLink size={18} aria-hidden />}
          onClick={() => setDialog('reissue')}
        >
          編集リンクを再発行
        </Button>
        {reveal ? (
          <Button
            variant='subtle'
            color='forest'
            leftSection={<IconEyeOff size={18} aria-hidden />}
            onClick={() => setReveal(false)}
          >
            個人情報を隠す
          </Button>
        ) : (
          <Button
            variant='subtle'
            color='forest'
            leftSection={<IconEye size={18} aria-hidden />}
            onClick={() => setDialog('reveal')}
          >
            個人情報を表示
          </Button>
        )}
      </Group>

      <Card padding='lg'>
        <Title order={2} fz='lg' mb='sm'>
          宿泊情報
        </Title>
        <SimpleGrid cols={{ base: 1, sm: 2, lg: 4 }} spacing='md'>
          <Field label='予定人数'>{d.planned_guest_count}名</Field>
          <Field label='登録人数'>
            {d.registered_count}名{d.remaining_count > 0 ? `（残り${d.remaining_count}名）` : ''}
          </Field>
          <Field label='到着予定時刻'>{d.estimated_arrival_time ?? '—'}</Field>
          <Field label='登録時の定員'>{d.capacity_snapshot}名</Field>
          <Field label='予約経路'>{d.booking_source ?? '—'}</Field>
          <Field label='予約番号'>{d.booking_reference ?? '—'}</Field>
          <Field label='送信日時'>{formatInstant(d.submitted_at, tz)}</Field>
          <Field label='本人確認日時'>
            {d.identity_verified_at ? formatInstant(d.identity_verified_at, tz) : '未確認'}
          </Field>
          <Field label='有効な編集リンク'>
            {d.active_edit_links > 0 && d.edit_link_expires_at
              ? `${d.active_edit_links}件（${formatInstant(d.edit_link_expires_at, tz)}まで）`
              : 'なし'}
          </Field>
          <Field label='同意した規約の版'>
            利用規約 {d.terms_version} / プライバシー {d.privacy_version}
          </Field>
        </SimpleGrid>
      </Card>

      <Stack gap='sm'>
        <Group justify='space-between'>
          <Title order={2} fz='lg'>
            宿泊者
          </Title>
          <Text size='sm' c='dimmed'>
            {d.revealed ? '個人情報を表示中（監査ログに記録済み）' : '住所・電話・メールは一部を隠して表示しています'}
          </Text>
        </Group>
        <Stack gap='sm' component='ul' style={{ listStyle: 'none', padding: 0, margin: 0 }}>
          {d.guests.map((g, i) => (
            <GuestCard key={g.public_id} guest={g} index={i} />
          ))}
        </Stack>
      </Stack>

      <ConfirmDialog
        opened={dialog === 'reveal'}
        title='個人情報を表示しますか？'
        confirmLabel='表示する'
        onClose={() => setDialog(null)}
        onConfirm={() => {
          setReveal(true);
          setDialog(null);
        }}
      >
        <Text>住所・電話番号・メールアドレスを表示します。表示した操作は監査ログに記録されます。</Text>
      </ConfirmDialog>
      <ConfirmDialog
        opened={dialog === 'verify'}
        title='本人確認を記録しますか？'
        confirmLabel='本人確認済みにする'
        loading={verify.isPending}
        onClose={() => setDialog(null)}
        onConfirm={() => verify.mutate(undefined, { onSettled: () => setDialog(null) })}
      >
        <Text>対面で宿泊者の本人確認を行った場合のみ記録してください。記録者と日時が保存されます。</Text>
      </ConfirmDialog>
      <ConfirmDialog
        opened={dialog === 'reissue'}
        title='編集リンクを再発行しますか？'
        confirmLabel='再発行する'
        danger
        loading={reissue.isPending}
        onClose={() => setDialog(null)}
        onConfirm={() =>
          reissue.mutate(undefined, {
            onSuccess: (r) => setLink(r),
            onSettled: () => setDialog(null)
          })
        }
      >
        <Text>これまでに発行した編集リンクはすべて無効になります。新しいリンクは一度だけ表示されます。</Text>
      </ConfirmDialog>
      {dialog === 'correct' ? <CorrectionForm detail={d} uid={uid} opened onClose={() => setDialog(null)} /> : null}
      <ReissueLinkDialog result={link} timezone={tz} onClose={() => setLink(null)} />
    </Stack>
  );
}

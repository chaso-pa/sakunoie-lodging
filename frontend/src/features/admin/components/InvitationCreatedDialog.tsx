'use client';

import { Button, CopyButton, Group, Modal, NativeSelect, Stack, Text, TextInput } from '@mantine/core';
import { IconCopy } from '@tabler/icons-react';
import { useEffect, useState } from 'react';
import { InfoBanner } from '@/components/InfoBanner';
import { paths } from '@/config/paths';
import type { CreateInvitationResult } from '@/features/admin/types/api';
import { formatDate, formatInstant } from '@/features/admin/utils/format';

const LOCALES = [
  { value: 'ja', label: '日本語' },
  { value: 'en', label: 'English' },
  { value: 'zh-CN', label: '简体中文' },
  { value: 'ko', label: '한국어' }
];

/**
 * Shows the new invitation URL once. The token lives only in this component's state (and the
 * parent's) and is dropped when the dialog closes; the server stores only its hash.
 */
export function InvitationCreatedDialog({
  result,
  timezone,
  onClose
}: {
  result: CreateInvitationResult | null;
  timezone: string;
  onClose: () => void;
}) {
  const [locale, setLocale] = useState('ja');
  const [origin, setOrigin] = useState('');
  useEffect(() => setOrigin(window.location.origin), []);
  const url = result
    ? paths.registration.wizard.getAbsoluteInviteHref(
        origin,
        locale,
        result.invitation.facility.code,
        result.invitation_token
      )
    : '';

  return (
    <Modal
      opened={result !== null}
      onClose={onClose}
      title={<Text fw={700}>招待リンクを作成しました</Text>}
      centered
      size='lg'
    >
      {result ? (
        <Stack gap='md'>
          <InfoBanner tone='warning' title='このリンクは今だけ表示されます'>
            閉じると二度と表示できません。コピーして、Airbnb
            のメッセージでゲストに送ってください。わからなくなった場合は、この招待を取り消して新しく作成してください。
          </InfoBanner>
          <Text size='sm'>
            宿泊日: {formatDate(result.invitation.check_in_date)} – {formatDate(result.invitation.check_out_date)}
            （ゲストは変更できません）
          </Text>
          <Text size='sm'>有効期限: {formatInstant(result.invitation.expires_at, timezone)}（施設の現地時刻）</Text>
          <NativeSelect
            label='リンクの表示言語'
            description='ゲストが読む言語を選んでください。'
            value={locale}
            onChange={(e) => setLocale(e.currentTarget.value)}
            data={LOCALES}
          />
          <TextInput label='招待リンク' value={url} readOnly onFocus={(e) => e.currentTarget.select()} />
          <Text size='sm' c='dimmed'>
            このリンクで登録できるのは1回だけです。登録後、ゲストは登録完了画面に表示される編集用リンクを使います。
          </Text>
          <Group justify='flex-end'>
            <CopyButton value={url}>
              {({ copied, copy }) => (
                <Button color='forest' onClick={copy} leftSection={<IconCopy size={18} aria-hidden />}>
                  {copied ? 'コピーしました' : 'リンクをコピー'}
                </Button>
              )}
            </CopyButton>
            <Button variant='default' onClick={onClose}>
              閉じる
            </Button>
          </Group>
        </Stack>
      ) : null}
    </Modal>
  );
}

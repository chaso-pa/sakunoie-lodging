'use client';

import { Button, CopyButton, Group, Modal, NativeSelect, Stack, Text, TextInput } from '@mantine/core';
import { useEffect, useState } from 'react';
import { InfoBanner } from '@/components/InfoBanner';
import { paths } from '@/config/paths';
import type { ReissueEditLinkResult } from '@/features/admin/types/api';
import { formatInstant } from '@/features/admin/utils/format';

const LOCALES = [
  { value: 'ja', label: '日本語' },
  { value: 'en', label: 'English' },
  { value: 'zh-CN', label: '简体中文' },
  { value: 'ko', label: '한국어' }
];

/**
 * Shows the new edit URL once. The token lives only in this component's state and is dropped when
 * the dialog closes; the server stores only its hash.
 */
export function ReissueLinkDialog({
  result,
  timezone,
  onClose
}: {
  result: ReissueEditLinkResult | null;
  timezone: string;
  onClose: () => void;
}) {
  const [locale, setLocale] = useState('ja');
  const [origin, setOrigin] = useState('');
  useEffect(() => setOrigin(window.location.origin), []);
  const url = result ? paths.registration.manage.getAbsoluteHref(origin, locale, result.edit_token) : '';

  return (
    <Modal opened={result !== null} onClose={onClose} title={<Text fw={700}>新しい編集リンク</Text>} centered size='lg'>
      {result ? (
        <Stack gap='md'>
          <InfoBanner tone='warning' title='このリンクは今だけ表示されます'>
            以前のリンクはすべて無効になりました。宿泊者本人に安全な方法で伝えてください。リンクを知っている人は登録内容の一部を変更できます。
          </InfoBanner>
          {result.expired ? (
            <InfoBanner tone='error' title='チェックアウト日を過ぎています'>
              このリンクは期限切れのため使えません。訂正は管理画面から行ってください。
            </InfoBanner>
          ) : (
            <Text size='sm'>有効期限: {formatInstant(result.edit_token_expires_at, timezone)}（施設の現地時刻）</Text>
          )}
          <NativeSelect
            label='リンクの表示言語'
            value={locale}
            onChange={(e) => setLocale(e.currentTarget.value)}
            data={LOCALES}
          />
          <TextInput label='編集リンク' value={url} readOnly onFocus={(e) => e.currentTarget.select()} />
          <Group justify='flex-end'>
            <CopyButton value={url}>
              {({ copied, copy }) => (
                <Button color='forest' onClick={copy}>
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

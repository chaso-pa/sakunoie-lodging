'use client';

import {
  Button,
  Checkbox,
  Divider,
  Group,
  Modal,
  NumberInput,
  Radio,
  Stack,
  Text,
  TextInput,
  Title
} from '@mantine/core';
import { type FormEvent, useState } from 'react';
import { InfoBanner } from '@/components/InfoBanner';
import { usePatchAdminRegistration } from '@/features/admin/api/registration-actions';
import type {
  AdminGuest,
  AdminPatchGuest,
  AdminPatchRegistration,
  AdminRegistrationDetail
} from '@/features/admin/types/api';
import { adminErrorMessage } from '@/features/admin/utils/error-message';
import { ApiError } from '@/lib/api-client';

type GuestDraft = {
  full_name: string;
  name_kana: string;
  address: string;
  phone: string;
  email: string;
  nationality: string;
  has_japan_address: boolean;
  passport_number: string;
};

const guestDraft = (g: AdminGuest): GuestDraft => ({
  full_name: g.full_name,
  name_kana: g.name_kana ?? '',
  // Masked values are never prefilled: an empty field means "unchanged".
  address: '',
  phone: '',
  email: '',
  nationality: g.nationality ?? '',
  has_japan_address: g.has_japan_address,
  passport_number: ''
});

/** Builds a PATCH body that contains only the fields the host changed. */
export const buildPatch = (
  d: AdminRegistrationDetail,
  stay: {
    check_in_date: string;
    check_out_date: string;
    arrival: string;
    planned: number;
    source: string;
    reference: string;
  },
  override: boolean,
  guests: Record<string, GuestDraft>,
  representative: string
): AdminPatchRegistration => {
  const body: AdminPatchRegistration = {};
  if (stay.check_in_date !== d.check_in_date) body.check_in_date = stay.check_in_date;
  if (stay.check_out_date !== d.check_out_date) body.check_out_date = stay.check_out_date;
  if (stay.arrival !== (d.estimated_arrival_time ?? '')) body.estimated_arrival_time = stay.arrival;
  if (stay.planned !== d.planned_guest_count) body.planned_guest_count = stay.planned;
  if (override) body.capacity_override = true;
  if (stay.source !== (d.booking_source ?? '')) body.booking_source = stay.source;
  if (stay.reference !== (d.booking_reference ?? '')) body.booking_reference = stay.reference;
  const patches: AdminPatchGuest[] = [];
  for (const g of d.guests) {
    const draft = guests[g.public_id];
    const p: AdminPatchGuest = { public_id: g.public_id };
    if (draft.full_name.trim() !== g.full_name) p.full_name = draft.full_name.trim();
    if (draft.name_kana.trim() !== (g.name_kana ?? '')) p.name_kana = draft.name_kana.trim();
    if (draft.address.trim()) p.address = draft.address.trim();
    if (draft.phone.trim()) p.phone = draft.phone.trim();
    if (draft.email.trim()) p.email = draft.email.trim();
    if (draft.nationality.trim() !== (g.nationality ?? '')) p.nationality = draft.nationality.trim();
    if (draft.has_japan_address !== g.has_japan_address) p.has_japan_address = draft.has_japan_address;
    if (draft.passport_number.trim()) p.passport_number = draft.passport_number.trim();
    if (representative === g.public_id && !g.is_representative) p.is_representative = true;
    if (Object.keys(p).length > 1) patches.push(p);
  }
  if (patches.length) body.guests = patches;
  return body;
};

export function CorrectionForm({
  detail,
  uid,
  opened,
  onClose
}: {
  detail: AdminRegistrationDetail;
  uid: string;
  opened: boolean;
  onClose: () => void;
}) {
  const mutation = usePatchAdminRegistration(uid, detail.public_id);
  const [stay, setStay] = useState({
    check_in_date: detail.check_in_date,
    check_out_date: detail.check_out_date,
    arrival: detail.estimated_arrival_time ?? '',
    planned: detail.planned_guest_count,
    source: detail.booking_source ?? '',
    reference: detail.booking_reference ?? ''
  });
  const [override, setOverride] = useState(false);
  const [guests, setGuests] = useState<Record<string, GuestDraft>>(() =>
    Object.fromEntries(detail.guests.map((g) => [g.public_id, guestDraft(g)]))
  );
  const [representative, setRepresentative] = useState(detail.guests.find((g) => g.is_representative)?.public_id ?? '');
  const [nothing, setNothing] = useState(false);

  const setGuest = (id: string, patch: Partial<GuestDraft>) =>
    setGuests((all) => ({ ...all, [id]: { ...all[id], ...patch } }));

  const submit = (e: FormEvent) => {
    e.preventDefault();
    const body = buildPatch(detail, stay, override, guests, representative);
    if (Object.keys(body).length === 0) {
      setNothing(true);
      return;
    }
    setNothing(false);
    mutation.mutate({ version: detail.version, body }, { onSuccess: onClose });
  };

  const error = mutation.error;
  return (
    <Modal opened={opened} onClose={onClose} title={<Text fw={700}>登録内容の訂正</Text>} size='xl'>
      <form onSubmit={submit} noValidate>
        <Stack gap='md'>
          <Text size='sm' c='dimmed'>
            変更した項目だけが送信され、項目名が監査ログに記録されます。住所・電話・メール・旅券番号は、変更する場合のみ入力してください。
          </Text>
          {error ? (
            <InfoBanner tone='error' announce='assertive'>
              {adminErrorMessage(error)}
              {error instanceof ApiError && error.details.length ? (
                <ul style={{ margin: '4px 0 0', paddingLeft: 20 }}>
                  {error.details.map((d) => (
                    <li key={d.location}>{d.location}</li>
                  ))}
                </ul>
              ) : null}
            </InfoBanner>
          ) : null}
          {nothing ? <InfoBanner tone='info'>変更された項目はありません。</InfoBanner> : null}

          <Title order={3} fz='lg'>
            宿泊情報
          </Title>
          <Group grow align='flex-start' wrap='wrap'>
            <TextInput
              label='チェックイン'
              type='date'
              value={stay.check_in_date}
              onChange={(e) => setStay({ ...stay, check_in_date: e.currentTarget.value })}
              miw={180}
            />
            <TextInput
              label='チェックアウト'
              type='date'
              value={stay.check_out_date}
              onChange={(e) => setStay({ ...stay, check_out_date: e.currentTarget.value })}
              miw={180}
            />
            <TextInput
              label='到着予定時刻'
              type='time'
              value={stay.arrival}
              onChange={(e) => setStay({ ...stay, arrival: e.currentTarget.value })}
              miw={140}
            />
          </Group>
          <Group grow align='flex-start' wrap='wrap'>
            <NumberInput
              label='予定人数'
              min={1}
              max={50}
              value={stay.planned}
              onChange={(v) => setStay({ ...stay, planned: typeof v === 'number' ? v : stay.planned })}
              description={`施設の定員: ${detail.facility.max_guests}名`}
              miw={160}
            />
            <TextInput
              label='予約経路'
              value={stay.source}
              maxLength={50}
              onChange={(e) => setStay({ ...stay, source: e.currentTarget.value })}
              miw={160}
            />
            <TextInput
              label='予約番号'
              value={stay.reference}
              maxLength={255}
              onChange={(e) => setStay({ ...stay, reference: e.currentTarget.value })}
              miw={160}
            />
          </Group>
          {stay.planned > detail.facility.max_guests ? (
            <Checkbox
              label={`定員超過を許可する（${detail.facility.max_guests}名を超える予定人数を保存）`}
              checked={override}
              onChange={(e) => setOverride(e.currentTarget.checked)}
            />
          ) : null}

          <Radio.Group label='代表者' value={representative} onChange={setRepresentative}>
            <Group mt='xs'>
              {detail.guests.map((g) => (
                <Radio key={g.public_id} value={g.public_id} label={g.full_name} />
              ))}
            </Group>
          </Radio.Group>

          {detail.guests.map((g, i) => {
            const draft = guests[g.public_id];
            return (
              <Stack key={g.public_id} gap='sm' component='fieldset' style={{ border: 0, padding: 0, margin: 0 }}>
                <Divider />
                <Text component='legend' fw={700}>
                  宿泊者 {i + 1}
                  {g.is_representative ? '（代表者）' : ''}
                </Text>
                <Group grow align='flex-start' wrap='wrap'>
                  <TextInput
                    label='氏名'
                    value={draft.full_name}
                    maxLength={255}
                    onChange={(e) => setGuest(g.public_id, { full_name: e.currentTarget.value })}
                    miw={200}
                  />
                  <TextInput
                    label='フリガナ'
                    value={draft.name_kana}
                    maxLength={255}
                    onChange={(e) => setGuest(g.public_id, { name_kana: e.currentTarget.value })}
                    miw={200}
                  />
                </Group>
                <TextInput
                  label='住所'
                  placeholder='変更する場合のみ入力'
                  value={draft.address}
                  maxLength={2000}
                  onChange={(e) => setGuest(g.public_id, { address: e.currentTarget.value })}
                />
                <Group grow align='flex-start' wrap='wrap'>
                  <TextInput
                    label='電話番号'
                    placeholder='変更する場合のみ入力'
                    value={draft.phone}
                    maxLength={50}
                    onChange={(e) => setGuest(g.public_id, { phone: e.currentTarget.value })}
                    miw={200}
                  />
                  <TextInput
                    label='メールアドレス'
                    placeholder='変更する場合のみ入力'
                    value={draft.email}
                    maxLength={255}
                    onChange={(e) => setGuest(g.public_id, { email: e.currentTarget.value })}
                    miw={200}
                  />
                </Group>
                <Checkbox
                  label='日本国内に住所がある'
                  checked={draft.has_japan_address}
                  onChange={(e) => setGuest(g.public_id, { has_japan_address: e.currentTarget.checked })}
                  description={
                    g.passport_last4 && draft.has_japan_address ? '保存すると旅券番号は削除されます' : undefined
                  }
                />
                {!draft.has_japan_address ? (
                  <Group grow align='flex-start' wrap='wrap'>
                    <TextInput
                      label='国籍'
                      value={draft.nationality}
                      maxLength={100}
                      onChange={(e) => setGuest(g.public_id, { nationality: e.currentTarget.value })}
                      miw={200}
                    />
                    <TextInput
                      label='旅券番号'
                      placeholder={
                        g.passport_last4
                          ? `変更する場合のみ入力（現在: 下4桁 ${g.passport_last4}）`
                          : '入力してください'
                      }
                      value={draft.passport_number}
                      maxLength={20}
                      autoComplete='off'
                      onChange={(e) => setGuest(g.public_id, { passport_number: e.currentTarget.value })}
                      miw={200}
                    />
                  </Group>
                ) : null}
              </Stack>
            );
          })}

          <Group justify='flex-end'>
            <Button variant='default' onClick={onClose} disabled={mutation.isPending}>
              キャンセル
            </Button>
            <Button type='submit' color='forest' loading={mutation.isPending}>
              訂正を保存
            </Button>
          </Group>
        </Stack>
      </form>
    </Modal>
  );
}

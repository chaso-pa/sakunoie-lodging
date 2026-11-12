'use client';

import { Button, Group, NativeSelect, TextInput } from '@mantine/core';
import { IconSearch } from '@tabler/icons-react';
import { type FormEvent, useEffect, useState } from 'react';
import type { AdminFacility, LedgerFilter } from '@/features/admin/types/api';

type Props = { facilities: AdminFacility[]; value: LedgerFilter; onChange: (f: LedgerFilter) => void };

/** Facility, check-in range and search. Applied on submit; stacks on mobile, inline on desktop. */
export function FacilityFilter({ facilities, value, onChange }: Props) {
  const [draft, setDraft] = useState(value);
  useEffect(() => setDraft(value), [value]);

  const submit = (e: FormEvent) => {
    e.preventDefault();
    onChange({ ...draft, page: 1 });
  };

  return (
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
          label='チェックイン（から）'
          type='date'
          value={draft.from ?? ''}
          onChange={(e) => setDraft({ ...draft, from: e.currentTarget.value || undefined })}
          w={{ base: '100%', sm: 180 }}
        />
        <TextInput
          label='チェックイン（まで）'
          type='date'
          value={draft.to ?? ''}
          onChange={(e) => setDraft({ ...draft, to: e.currentTarget.value || undefined })}
          w={{ base: '100%', sm: 180 }}
        />
        <TextInput
          label='氏名・予約番号で検索'
          value={draft.q ?? ''}
          maxLength={100}
          onChange={(e) => setDraft({ ...draft, q: e.currentTarget.value || undefined })}
          w={{ base: '100%', sm: 240 }}
        />
        <Button type='submit' color='forest' leftSection={<IconSearch size={18} aria-hidden />}>
          検索
        </Button>
      </Group>
    </form>
  );
}

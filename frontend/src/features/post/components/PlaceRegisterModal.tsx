'use client';

import { Button, Modal, TextInput } from '@mantine/core';
import { useMediaQuery } from '@mantine/hooks';
import { useTranslations } from 'next-intl';
import { useState } from 'react';
import shared from '@/components/Community.module.scss';
import { placeAddressMaxLength, placeNameMaxLength } from '@/features/post/api/register-place';
import classes from '@/features/post/components/Place.module.scss';
import type { LatLng, PlaceSelection } from '@/features/post/types/place';

export type RegisterDraft = { name: string; address: string; position: LatLng | null };

type PlaceRegisterModalProps = {
  /** null closes the modal. The draft lives in the picker so it survives a trip to the map. */
  draft: RegisterDraft | null;
  onChange: (draft: RegisterDraft) => void;
  onClose: () => void;
  onPickPosition: () => void;
  onSubmit: (selection: PlaceSelection) => void;
};

export const formatPosition = (position: LatLng) => `${position.latitude.toFixed(5)}, ${position.longitude.toFixed(5)}`;

/**
 * Hand-entered place for what search cannot find. Only the name is required. A position is sent only when
 * the member put a pin on the map; none is ever made up.
 */
export function PlaceRegisterModal({ draft, onChange, onClose, onPickPosition, onSubmit }: PlaceRegisterModalProps) {
  const t = useTranslations('community.compose.place.register');
  const tCommon = useTranslations('community.common');
  const isPhone = useMediaQuery('(max-width: 520px)');
  const [error, setError] = useState('');

  const submit = (event: React.FormEvent) => {
    event.preventDefault();
    // React events bubble through portals: without this the composer's own form would submit too.
    event.stopPropagation();
    if (!draft) {
      return;
    }
    const name = draft.name.trim();
    if (!name) {
      setError(t('errors.nameRequired'));
      return;
    }
    setError('');
    onSubmit({ kind: 'manual', name, address: draft.address.trim(), position: draft.position });
  };

  return (
    <Modal
      opened={draft !== null}
      onClose={onClose}
      title={t('title')}
      fullScreen={isPhone}
      size={480}
      centered
      closeButtonProps={{ 'aria-label': tCommon('cancel'), size: 'lg' }}
    >
      {draft && (
        <form onSubmit={submit} noValidate>
          <TextInput
            label={t('nameLabel')}
            placeholder={t('namePlaceholder')}
            size='md'
            radius='sm'
            classNames={{ label: shared.fieldLabel }}
            required
            data-autofocus
            maxLength={placeNameMaxLength}
            value={draft.name}
            error={error || undefined}
            onChange={(event) => onChange({ ...draft, name: event.currentTarget.value })}
          />
          <TextInput
            className={shared.field}
            label={t('addressLabel')}
            placeholder={t('addressPlaceholder')}
            size='md'
            radius='sm'
            classNames={{ label: shared.fieldLabel }}
            maxLength={placeAddressMaxLength}
            value={draft.address}
            onChange={(event) => onChange({ ...draft, address: event.currentTarget.value })}
          />
          <div className={shared.fieldLabel} id='place-register-position'>
            {t('positionLabel')}
          </div>
          <output className={shared.muted} style={{ display: 'block' }} aria-labelledby='place-register-position'>
            {draft.position ? t('positionSet', { position: formatPosition(draft.position) }) : t('positionNone')}
          </output>
          <div className={classes.positionRow}>
            <Button variant='light' size='sm' onClick={onPickPosition}>
              🗺 {draft.position ? t('changePosition') : t('pickPosition')}
            </Button>
            {draft.position && (
              <Button variant='default' size='sm' onClick={() => onChange({ ...draft, position: null })}>
                {t('removePosition')}
              </Button>
            )}
          </div>
          <div className={shared.note} style={{ marginTop: 16 }}>
            <strong>{t('noticeTitle')}</strong>
            {t('noticeBody')}
          </div>
          <Button type='submit' fullWidth mt={16} size='lg'>
            {t('submit')}
          </Button>
          <Button variant='default' fullWidth mt={8} size='md' onClick={onClose}>
            {tCommon('cancel')}
          </Button>
        </form>
      )}
    </Modal>
  );
}

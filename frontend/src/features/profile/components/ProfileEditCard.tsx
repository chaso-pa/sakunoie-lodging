'use client';

import { Button, FileButton, NativeSelect, TextInput } from '@mantine/core';
import { notifications } from '@mantine/notifications';
import { useLocale, useTranslations } from 'next-intl';
import { useEffect, useState, useTransition } from 'react';
import shared from '@/components/Community.module.scss';
import { MemberAvatar } from '@/components/MemberAvatar';
import { nicknameMaxLength, useUpdateProfile } from '@/features/profile/api/update-profile';
import classes from '@/features/profile/components/Profile.module.scss';
import { isLocale, type Locale, localeLabels, locales } from '@/i18n/config';
import { setLocale } from '@/i18n/set-locale';
import { getApiErrorStatus, isUploadTooLarge } from '@/lib/api-client';
import type { Member } from '@/types/member';
import { downscaleImage, ImageError } from '@/utils/downscale-image';

/** Nickname, picture and display language of the signed-in member. Saved through PATCH /members/me. */
export function ProfileEditCard({ member }: { member: Member }) {
  const t = useTranslations('community.profile');
  const tCommon = useTranslations('community.common');
  const tErrors = useTranslations('errors');
  const locale = useLocale();
  const updateProfile = useUpdateProfile();
  const [switching, startTransition] = useTransition();

  const [nickname, setNickname] = useState(member.nickname);
  const [language, setLanguage] = useState<Locale>(locale);
  const [avatar, setAvatar] = useState<File | null>(null);
  const [preview, setPreview] = useState<string | null>(null);
  const [processing, setProcessing] = useState(false);
  const [error, setError] = useState('');

  useEffect(() => {
    if (!avatar) {
      setPreview(null);
      return;
    }
    const url = URL.createObjectURL(avatar);
    setPreview(url);
    return () => URL.revokeObjectURL(url);
  }, [avatar]);

  const selectAvatar = async (file: File | null) => {
    if (!file) {
      return;
    }
    setError('');
    setProcessing(true);
    try {
      setAvatar(await downscaleImage(file));
    } catch (failure) {
      const problem = failure instanceof ImageError ? failure.problem : 'unreadable';
      setError(
        problem === 'type'
          ? tCommon('imageType')
          : problem === 'tooLarge'
            ? tCommon('fileTooLarge')
            : tCommon('imageUnreadable')
      );
    } finally {
      setProcessing(false);
    }
  };

  const submit = (event: React.FormEvent) => {
    event.preventDefault();
    const name = nickname.trim();
    if (!name) {
      setError(t('errors.nicknameRequired'));
      return;
    }
    // Count characters, not UTF-16 units, so emoji are not double counted.
    if ([...name].length > nicknameMaxLength) {
      setError(t('errors.nicknameTooLong'));
      return;
    }
    setError('');
    updateProfile.mutate(
      { nickname: name, language, ...(avatar && { avatar }) },
      {
        onSuccess: () => {
          setAvatar(null);
          notifications.show({ message: t('saved') });
          if (language !== locale) {
            startTransition(() => setLocale(language));
          }
        },
        onError: (failure) => {
          const status = getApiErrorStatus(failure);
          if (isUploadTooLarge(failure)) {
            setError(tCommon('fileTooLarge'));
          } else if (status === undefined) {
            setError(tErrors('network'));
          } else if (status === 422 || status === 400) {
            setError(t('errors.invalid'));
          } else if (status === 429) {
            setError(tErrors('rateLimited'));
          } else {
            setError(tErrors('generic'));
          }
        }
      }
    );
  };

  return (
    <div className={shared.card}>
      <form className={shared.cardBody} onSubmit={submit} noValidate>
        <h2 className={shared.h2}>{t('cardTitle')}</h2>
        <div className={classes.identity}>
          <MemberAvatar src={preview ?? member.avatar_url} size={64} alt={t('avatarAlt')} />
          <div style={{ minWidth: 0 }}>
            <strong>{member.nickname}</strong>
            <div className={shared.muted}>{t('joinedAs')}</div>
          </div>
        </div>
        <div className={classes.avatarButtons}>
          <FileButton onChange={selectAvatar} accept='image/*'>
            {(props) => (
              <Button {...props} variant='light' size='sm' loading={processing}>
                📷 {t('chooseAvatar')}
              </Button>
            )}
          </FileButton>
          {avatar && (
            <Button variant='default' size='sm' onClick={() => setAvatar(null)}>
              {t('cancelAvatar')}
            </Button>
          )}
        </div>
        <div className={shared.note} style={{ margin: '10px 0 14px' }}>
          <strong>{t('memberTitle')}</strong>
          {t('memberBody')}
        </div>
        <TextInput
          className={shared.field}
          label={t('nicknameLabel')}
          size='md'
          radius='sm'
          classNames={{ label: shared.fieldLabel }}
          autoComplete='nickname'
          maxLength={nicknameMaxLength * 2}
          value={nickname}
          onChange={(event) => setNickname(event.currentTarget.value)}
        />
        <NativeSelect
          className={shared.field}
          label={t('languageLabel')}
          size='md'
          radius='sm'
          classNames={{ label: shared.fieldLabel }}
          value={language}
          onChange={(event) => {
            const next = event.currentTarget.value;
            if (isLocale(next)) {
              setLanguage(next);
            }
          }}
          data={locales.map((value) => ({ value, label: localeLabels[value] }))}
        />
        <p className={shared.formError} role='alert'>
          {error}
        </p>
        <Button type='submit' fullWidth size='md' loading={updateProfile.isPending || switching} disabled={processing}>
          {t('save')}
        </Button>
      </form>
    </div>
  );
}

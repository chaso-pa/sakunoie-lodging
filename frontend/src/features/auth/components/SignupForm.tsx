'use client';

import { Avatar, Button, FileButton, TextInput, UnstyledButton } from '@mantine/core';
import { useForm } from '@mantine/form';
import { useLocale, useTranslations } from 'next-intl';
import { useEffect, useState } from 'react';
import { LanguageSwitcher } from '@/components/LanguageSwitcher';
import {
  avatarMaxBytes,
  avatarMimeTypes,
  nicknameMaxLength,
  type SignupResponse,
  useSignup
} from '@/features/auth/api/signup';
import classes from '@/features/auth/components/Auth.module.scss';
import { AuthCard } from '@/features/auth/components/AuthCard';
import { getApiErrorStatus } from '@/lib/api-client';

type SignupFormProps = {
  onSignedUp: (result: SignupResponse) => void;
  onBackToInvite: () => void;
};

export function SignupForm({ onSignedUp, onBackToInvite }: SignupFormProps) {
  const t = useTranslations('auth.signup');
  const tAuth = useTranslations('auth');
  const tErrors = useTranslations('errors');
  const locale = useLocale();
  const signup = useSignup();

  const [avatar, setAvatar] = useState<File | null>(null);
  const [avatarError, setAvatarError] = useState('');
  const [preview, setPreview] = useState<string | null>(null);

  useEffect(() => {
    if (!avatar) {
      setPreview(null);
      return;
    }
    const url = URL.createObjectURL(avatar);
    setPreview(url);
    return () => URL.revokeObjectURL(url);
  }, [avatar]);

  const form = useForm({
    mode: 'uncontrolled',
    initialValues: { nickname: '' },
    validate: {
      nickname: (value) => {
        const nickname = value.trim();
        if (!nickname) {
          return t('errors.nicknameRequired');
        }
        // Count characters, not UTF-16 units, so emoji are not double counted.
        return [...nickname].length > nicknameMaxLength ? t('errors.nicknameTooLong') : null;
      }
    }
  });

  const selectAvatar = (file: File | null) => {
    if (!file) {
      return;
    }
    if (!avatarMimeTypes.includes(file.type)) {
      setAvatarError(t('errors.avatarType'));
      return;
    }
    if (file.size > avatarMaxBytes) {
      setAvatarError(t('errors.avatarTooLarge'));
      return;
    }
    setAvatarError('');
    setAvatar(file);
  };

  const status = signup.isError ? getApiErrorStatus(signup.error) : undefined;
  const inviteExpired = status === 403;

  const errorMessage = () => {
    if (avatarError) {
      return avatarError;
    }
    if (!signup.isError) {
      return '';
    }
    if (inviteExpired) {
      return t('errors.inviteNotVerified');
    }
    if (status === 413) {
      return t('errors.avatarTooLarge');
    }
    if (status === 422 || status === 400) {
      return t('errors.invalid');
    }
    if (status === 429) {
      return tErrors('rateLimited');
    }
    return status === undefined ? tErrors('network') : tErrors('generic');
  };

  return (
    <AuthCard icon='🌿' eyebrow={t('eyebrow')} title={t('title')} step={2}>
      <p className={classes.muted}>{t('description')}</p>
      <form
        onSubmit={form.onSubmit(({ nickname }) =>
          signup.mutate({ nickname: nickname.trim(), language: locale, avatar }, { onSuccess: onSignedUp })
        )}
      >
        <div className={classes.avatarPicker}>
          <Avatar className={classes.avatar} src={preview} alt={t('avatarAlt')} size={88} radius='50%'>
            🙂
          </Avatar>
          <FileButton onChange={selectAvatar} accept={avatarMimeTypes.join(',')}>
            {(props) => (
              <Button {...props} variant='light' size='sm'>
                {t('chooseAvatar')}
              </Button>
            )}
          </FileButton>
          {avatar ? (
            <UnstyledButton className={classes.muted} onClick={() => setAvatar(null)}>
              {t('removeAvatar')}
            </UnstyledButton>
          ) : (
            <span className={classes.muted}>{t('avatarOptional')}</span>
          )}
        </div>
        <TextInput
          key={form.key('nickname')}
          {...form.getInputProps('nickname')}
          label={t('nicknameLabel')}
          placeholder={t('nicknamePlaceholder')}
          maxLength={nicknameMaxLength * 2}
          autoComplete='nickname'
          size='md'
          radius='sm'
          classNames={{ label: classes.fieldLabel }}
        />
        <LanguageSwitcher
          label={t('languageLabel')}
          size='md'
          fullWidth
          mt={12}
          classNames={{ label: classes.fieldLabel }}
        />
        <p className={classes.error} role='alert'>
          {errorMessage()}
        </p>
        {inviteExpired ? (
          <Button size='md' fullWidth mt={10} variant='light' onClick={onBackToInvite}>
            {tAuth('backToInvite')}
          </Button>
        ) : (
          <Button type='submit' size='md' fullWidth mt={10} loading={signup.isPending}>
            {t('submit')}
          </Button>
        )}
      </form>
      <div className={classes.note}>{t('note')}</div>
    </AuthCard>
  );
}

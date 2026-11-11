'use client';

import { Group, Radio, Stack, Textarea, TextInput } from '@mantine/core';
import { useTranslations } from 'next-intl';
import { InfoBanner } from '@/components/InfoBanner';
import { OptionalLabel } from '@/features/registration/components/OptionalLabel';
import { type GuestFormValues, requiresPassport } from '@/features/registration/types/schemas';
import { fieldId, type PathForm } from '@/features/registration/utils/form-paths';

type GuestFieldsProps = {
  form: PathForm;
  /** Form path of the guest object, e.g. `representative` or `companions.1`. */
  path: string;
  guest: GuestFormValues;
};

export function GuestFields({ form, path, guest }: GuestFieldsProps) {
  const t = useTranslations();
  const p = (field: keyof GuestFormValues) => `${path}.${field}`;
  const input = (field: keyof GuestFormValues) => ({ id: fieldId(p(field)), ...form.getInputProps(p(field)) });
  const japanAddress = form.getInputProps(p('hasJapanAddress'));

  return (
    <Stack gap='md'>
      <TextInput label={t('guest.fullName')} withAsterisk autoComplete='off' {...input('fullName')} />
      <TextInput label={<OptionalLabel label={t('guest.nameKana')} />} autoComplete='off' {...input('nameKana')} />
      <Textarea label={t('guest.address')} withAsterisk autosize minRows={2} autoComplete='off' {...input('address')} />
      <TextInput label={<OptionalLabel label={t('guest.phone')} />} type='tel' autoComplete='off' {...input('phone')} />
      <TextInput
        label={<OptionalLabel label={t('guest.email')} />}
        type='email'
        autoComplete='off'
        {...input('email')}
      />
      <Radio.Group
        id={fieldId(p('hasJapanAddress'))}
        label={t('guest.hasJapanAddress')}
        withAsterisk
        error={japanAddress.error}
        value={guest.hasJapanAddress === null ? '' : guest.hasJapanAddress ? 'yes' : 'no'}
        onChange={(value) => form.setFieldValue(p('hasJapanAddress'), value === 'yes')}
      >
        <Group mt='xs' gap='xl'>
          <Radio value='yes' label={t('common.yes')} size='md' />
          <Radio value='no' label={t('common.no')} size='md' />
        </Group>
      </Radio.Group>
      {requiresPassport(guest) ? (
        <Stack gap='md'>
          <InfoBanner tone='info'>{t('guest.passportHelp')}</InfoBanner>
          <TextInput label={t('guest.nationality')} withAsterisk autoComplete='off' {...input('nationality')} />
          <TextInput
            label={t('guest.passportNumber')}
            description={t('guest.passportPrivacy')}
            withAsterisk
            autoComplete='off'
            autoCapitalize='characters'
            spellCheck={false}
            {...input('passportNumber')}
          />
        </Stack>
      ) : null}
    </Stack>
  );
}

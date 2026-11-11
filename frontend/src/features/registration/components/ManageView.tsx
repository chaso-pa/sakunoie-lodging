'use client';

import { Badge, Button, Card, Container, Group, Loader, Stack, Text, Title } from '@mantine/core';
import { IconLock } from '@tabler/icons-react';
import { useQueryClient } from '@tanstack/react-query';
import { useLocale, useTranslations } from 'next-intl';
import { useCallback, useEffect, useState } from 'react';
import { InfoBanner } from '@/components/InfoBanner';
import { manageSummaryQueryKey, useManageSummary } from '@/features/registration/api/get-manage-summary';
import { AddCompanionsForm } from '@/features/registration/components/AddCompanionsForm';
import { ApiErrorBanner } from '@/features/registration/components/ApiErrorBanner';
import { CompanionEditForm } from '@/features/registration/components/CompanionEditForm';
import { RegistrationStatus } from '@/features/registration/components/RegistrationStatus';
import { StayEditForm } from '@/features/registration/components/StayEditForm';
import { consumeTokenFragment, manageTokenStore } from '@/features/registration/stores/manage-token-store';
import type { ManageGuest, ManageSummary } from '@/features/registration/types/api';
import { formatCivilDate, formatInstant } from '@/features/registration/utils/format';
import { ApiError } from '@/lib/api-client';

type Phase = 'init' | 'ready' | 'missing' | 'finished';
type AuthFailure = 'invalid' | 'expired';
type Notice = 'saved' | 'added' | 'precondition' | null;

export function ManageView() {
  const t = useTranslations('manage');
  const queryClient = useQueryClient();
  const [phase, setPhase] = useState<Phase>('init');
  const [authFailure, setAuthFailure] = useState<AuthFailure | null>(null);
  const [notice, setNotice] = useState<Notice>(null);
  const summary = useManageSummary(phase === 'ready' && authFailure === null);

  useEffect(() => {
    // Fragment → sessionStorage, then strip it from the address bar (plan Decision C).
    consumeTokenFragment();
    setPhase(manageTokenStore.get() ? 'ready' : 'missing');
  }, []);

  /** 401/410 end the session; 412 reloads the latest version. Returns true when handled here. */
  const handleError = useCallback(
    (error: unknown): boolean => {
      if (!(error instanceof ApiError)) return false;
      if (error.status === 401 || error.status === 410) {
        manageTokenStore.clear();
        queryClient.removeQueries({ queryKey: manageSummaryQueryKey });
        setAuthFailure(error.status === 410 ? 'expired' : 'invalid');
        return true;
      }
      if (error.code === 'precondition_failed') {
        setNotice('precondition');
        void queryClient.invalidateQueries({ queryKey: manageSummaryQueryKey });
        return true;
      }
      return false;
    },
    [queryClient]
  );

  useEffect(() => {
    if (summary.error) handleError(summary.error);
  }, [summary.error, handleError]);

  const finish = () => {
    manageTokenStore.clear();
    queryClient.removeQueries({ queryKey: manageSummaryQueryKey });
    setPhase('finished');
  };

  let body: React.ReactNode;
  if (authFailure === 'expired') {
    body = <Recovery title={t('expiredTitle')} body={t('expiredBody')} />;
  } else if (authFailure === 'invalid') {
    body = <Recovery title={t('invalidTitle')} body={t('invalidBody')} />;
  } else if (phase === 'missing') {
    body = <Recovery title={t('sessionLostTitle')} body={t('sessionLostBody')} />;
  } else if (phase === 'finished') {
    body = (
      <InfoBanner tone='success' title={t('finishedTitle')} announce='polite'>
        <Text size='sm'>{t('finishedBody')}</Text>
      </InfoBanner>
    );
  } else if (phase === 'init' || summary.isPending) {
    body = <Loader aria-label={t('title')} />;
  } else if (summary.isError) {
    body = (
      <Stack gap='sm'>
        <ApiErrorBanner error={summary.error} />
        <div>
          <Button variant='outline' onClick={() => summary.refetch()}>
            <RetryLabel />
          </Button>
        </div>
      </Stack>
    );
  } else {
    body = (
      <ManageContent
        summary={summary.data}
        notice={notice}
        setNotice={setNotice}
        onError={handleError}
        onFinish={finish}
      />
    );
  }

  return (
    <Container size={760} py={{ base: 24, md: 48 }}>
      <Stack gap='lg'>
        <Title order={1} fz={{ base: 26, md: 32 }}>
          {t('title')}
        </Title>
        {body}
      </Stack>
    </Container>
  );
}

function RetryLabel() {
  const t = useTranslations('common');
  return <>{t('retry')}</>;
}

function Recovery({ title, body }: { title: string; body: string }) {
  return (
    <InfoBanner tone='warning' title={title} announce='assertive'>
      <Text size='sm'>{body}</Text>
    </InfoBanner>
  );
}

function Field({ label, value }: { label: string; value: string }) {
  return (
    <Group gap='xs' wrap='nowrap' align='flex-start'>
      <Text size='sm' c='dimmed' miw={140}>
        {label}
      </Text>
      <Text size='sm' style={{ overflowWrap: 'anywhere' }}>
        {value}
      </Text>
    </Group>
  );
}

type ManageContentProps = {
  summary: ManageSummary;
  notice: Notice;
  setNotice: (notice: Notice) => void;
  onError: (error: unknown) => boolean;
  onFinish: () => void;
};

function ManageContent({ summary, notice, setNotice, onError, onFinish }: ManageContentProps) {
  const t = useTranslations();
  const locale = useLocale();
  const [editingStay, setEditingStay] = useState(false);
  const [editingGuest, setEditingGuest] = useState<string | null>(null);
  const notSet = t('common.notSet');

  const sourceLabel = (source: string | null) => {
    if (!source) return notSet;
    if (source === 'airbnb') return t('registration.stay.bookingSourceAirbnb');
    if (source === 'other') return t('registration.stay.bookingSourceOther');
    return source;
  };

  const noticeBanner =
    notice === 'precondition' ? (
      <InfoBanner tone='warning' announce='polite'>
        {t('errors.precondition_failed')}
      </InfoBanner>
    ) : notice ? (
      <InfoBanner tone='success' announce='polite'>
        {notice === 'added' ? t('manage.added') : t('manage.saved')}
      </InfoBanner>
    ) : null;

  return (
    <Stack gap='lg'>
      {noticeBanner}

      <Card padding='lg' component='section' aria-labelledby='manage-stay-title'>
        <Stack gap='sm'>
          <Title order={2} fz='xl' id='manage-stay-title'>
            {t('manage.stayTitle')}
          </Title>
          <Field label={t('manage.facility')} value={summary.facility.name} />
          <Field
            label={t('manage.dates')}
            value={`${formatCivilDate(summary.check_in_date, locale)} – ${formatCivilDate(summary.check_out_date, locale)}`}
          />
          <Field label={t('manage.arrival')} value={summary.estimated_arrival_time ?? notSet} />
          <Field label={t('manage.plannedCount')} value={String(summary.planned_guest_count)} />
          <Field label={t('manage.bookingSource')} value={sourceLabel(summary.booking_source)} />
          <Field label={t('manage.bookingReference')} value={summary.booking_reference ?? notSet} />
          <RegistrationStatus
            registered={summary.registered_count}
            planned={summary.planned_guest_count}
            remaining={summary.remaining_count}
            isComplete={summary.is_complete}
          />
          <Text size='sm' c='dimmed'>
            {t('manage.expiresAt', {
              date: formatInstant(summary.edit_expires_at, locale, summary.facility.timezone)
            })}
          </Text>
          {editingStay ? (
            <StayEditForm
              summary={summary}
              onError={onError}
              onCancel={() => setEditingStay(false)}
              onDone={() => {
                setEditingStay(false);
                setNotice('saved');
              }}
            />
          ) : (
            <div>
              <Button
                variant='outline'
                onClick={() => {
                  setNotice(null);
                  setEditingStay(true);
                }}
              >
                {t('manage.editStay')}
              </Button>
            </div>
          )}
        </Stack>
      </Card>

      <Card padding='lg' component='section' aria-labelledby='manage-guests-title'>
        <Stack gap='md'>
          <Title order={2} fz='xl' id='manage-guests-title'>
            {t('manage.guestsTitle')}
          </Title>
          {summary.guests.map((guest) => (
            <GuestItem
              key={guest.public_id}
              guest={guest}
              editing={editingGuest === guest.public_id}
              onEdit={() => {
                setNotice(null);
                setEditingGuest(guest.public_id);
              }}
              editForm={
                <CompanionEditForm
                  guest={guest}
                  summary={summary}
                  onError={onError}
                  onCancel={() => setEditingGuest(null)}
                  onDone={() => {
                    setEditingGuest(null);
                    setNotice('saved');
                  }}
                />
              }
            />
          ))}
        </Stack>
      </Card>

      <Card padding='lg' component='section' aria-labelledby='manage-add-title'>
        <Stack gap='md'>
          <Title order={2} fz='xl' id='manage-add-title'>
            {t('manage.addTitle')}
          </Title>
          {summary.remaining_count > 0 ? (
            <AddCompanionsForm summary={summary} onError={onError} onAdded={() => setNotice('added')} />
          ) : (
            <InfoBanner tone='success'>{t('manage.full')}</InfoBanner>
          )}
        </Stack>
      </Card>

      <div>
        <Button variant='default' onClick={onFinish}>
          {t('manage.finish')}
        </Button>
      </div>
    </Stack>
  );
}

type GuestItemProps = { guest: ManageGuest; editing: boolean; onEdit: () => void; editForm: React.ReactNode };

function GuestItem({ guest, editing, onEdit, editForm }: GuestItemProps) {
  const t = useTranslations();
  const notSet = t('common.notSet');
  return (
    <Card padding='md' bg='var(--sakunoie-surface-muted)' withBorder={false}>
      <Stack gap={6}>
        <Group justify='space-between'>
          <Badge variant='outline' color={guest.is_representative ? 'forest' : 'gray'} size='lg'>
            {guest.is_representative ? t('manage.representative') : t('manage.companion')}
          </Badge>
          {guest.editable && !editing ? (
            <Button variant='subtle' size='sm' onClick={onEdit}>
              {t('manage.correct')}
            </Button>
          ) : null}
        </Group>
        <Field label={t('guest.fullName')} value={guest.full_name_masked} />
        <Field label={t('guest.nameKana')} value={guest.name_kana_masked ?? notSet} />
        <Field label={t('guest.phone')} value={guest.phone_masked ?? notSet} />
        <Field label={t('guest.email')} value={guest.email_masked ?? notSet} />
        <Text size='sm'>{guest.has_japan_address ? t('manage.japanAddressYes') : t('manage.japanAddressNo')}</Text>
        {guest.passport_last4 ? (
          <Text size='sm'>{t('manage.passportLast4', { last4: guest.passport_last4 })}</Text>
        ) : null}
        {guest.is_representative ? (
          <Group gap={6} wrap='nowrap' align='flex-start'>
            <IconLock size={16} aria-hidden style={{ flexShrink: 0, marginTop: 3 }} />
            <Text size='sm' c='dimmed'>
              {t('manage.representativeReadOnly')}
            </Text>
          </Group>
        ) : null}
        {editing ? editForm : null}
      </Stack>
    </Card>
  );
}

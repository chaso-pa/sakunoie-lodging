'use client';

import { Button, Container, Group, Loader, Stack, Text, Title } from '@mantine/core';
import { useForm } from '@mantine/form';
import { useTranslations } from 'next-intl';
import { useEffect, useRef, useState } from 'react';
import { InfoBanner } from '@/components/InfoBanner';
import { ValidationSummary, type ValidationSummaryItem } from '@/components/ValidationSummary';
import { paths } from '@/config/paths';
import { useCreateRegistration } from '@/features/registration/api/create-registration';
import { useFacilityPublic } from '@/features/registration/api/get-facility-public';
import { ApiErrorBanner } from '@/features/registration/components/ApiErrorBanner';
import { CompanionsStep } from '@/features/registration/components/CompanionsStep';
import { FormStepper } from '@/features/registration/components/FormStepper';
import { RepresentativeStep } from '@/features/registration/components/RepresentativeStep';
import { PartialWarning, ReviewConsentStep } from '@/features/registration/components/ReviewConsentStep';
import { StayStep } from '@/features/registration/components/StayStep';
import { useWizardStore } from '@/features/registration/stores/wizard-store';
import type { FacilityPublic } from '@/features/registration/types/api';
import {
  emptyGuest,
  emptyWizardValues,
  errorsForStep,
  type FormErrorMap,
  stepOfPath,
  type ValidationKey,
  validateWizard,
  WIZARD_STEPS,
  type WizardFormValues
} from '@/features/registration/types/schemas';
import { errorMessageKey, wizardPathFromLocation } from '@/features/registration/utils/api-errors';
import { buildCreatePayload } from '@/features/registration/utils/build-payload';
import { fieldId } from '@/features/registration/utils/form-paths';
import { useRouter } from '@/i18n/navigation';
import { ApiError } from '@/lib/api-client';

type RegistrationWizardProps = { facilityCode: string };

export function RegistrationWizard({ facilityCode }: RegistrationWizardProps) {
  const t = useTranslations('registration');
  const facility = useFacilityPublic(facilityCode);
  const open = useWizardStore((s) => s.open);
  const alreadySubmitted = useWizardStore((s) => s.alreadySubmitted);
  const [ready, setReady] = useState(false);

  useEffect(() => {
    open(facilityCode);
    setReady(true);
  }, [open, facilityCode]);

  let body: React.ReactNode;
  if (!ready || facility.isPending) {
    body = <Loader aria-label={t('title')} />;
  } else if (facility.isError) {
    const unavailable = facility.error instanceof ApiError && facility.error.status === 404;
    body = unavailable ? (
      <InfoBanner tone='warning' title={t('facilityUnavailableTitle')} announce='assertive'>
        <FacilityUnavailableText />
      </InfoBanner>
    ) : (
      <Stack gap='sm'>
        <ApiErrorBanner error={facility.error} />
        <div>
          <Button variant='outline' onClick={() => facility.refetch()}>
            <RetryLabel />
          </Button>
        </div>
      </Stack>
    );
  } else if (alreadySubmitted) {
    body = <AlreadySubmitted />;
  } else {
    body = <WizardForm facility={facility.data} />;
  }

  return (
    <Container size={760} py={{ base: 24, md: 48 }}>
      <Stack gap='lg'>
        <Stack gap='xs'>
          <Title order={1} fz={{ base: 26, md: 32 }}>
            {t('title')}
          </Title>
          <Text c='dimmed'>{t('intro')}</Text>
        </Stack>
        {body}
      </Stack>
    </Container>
  );
}

function FacilityUnavailableText() {
  const t = useTranslations('errors');
  return <Text size='sm'>{t('facility_unavailable')}</Text>;
}

function RetryLabel() {
  const t = useTranslations('common');
  return <>{t('retry')}</>;
}

function AlreadySubmitted() {
  const t = useTranslations('registration');
  const startNew = useWizardStore((s) => s.startNew);
  return (
    <Stack gap='md'>
      <InfoBanner tone='warning' title={t('alreadySubmittedTitle')} announce='assertive'>
        <Text size='sm'>{t('alreadySubmittedBody')}</Text>
      </InfoBanner>
      <div>
        <Button variant='outline' onClick={startNew}>
          {t('startNew')}
        </Button>
      </div>
    </Stack>
  );
}

function WizardForm({ facility }: { facility: FacilityPublic }) {
  const t = useTranslations();
  const router = useRouter();
  const store = useWizardStore();
  const mutation = useCreateRegistration();
  const headingRef = useRef<HTMLHeadingElement>(null);
  const [summary, setSummary] = useState<{ items: ValidationSummaryItem[]; focusKey: number } | null>(null);

  const form = useForm<WizardFormValues>({
    mode: 'controlled',
    initialValues: store.draft ?? emptyWizardValues(),
    onValuesChange: (values) => store.saveDraft(values)
  });
  const values = form.getValues();
  const step = store.step;
  const planned = Number(values.stay.plannedGuestCount);
  const maxCompanions = Number.isInteger(planned) && planned > 0 ? planned - 1 : 0;
  const stepLabels = WIZARD_STEPS.map((s) => t(`registration.steps.${s}`));

  const translate = (key: ValidationKey) => t(`validation.${key}`, { max: facility.max_guests });

  const labelForPath = (path: string): string => {
    const [head, second, third] = path.split('.');
    const guestLabel = (field: string) => t(`guest.${field}` as 'guest.fullName');
    if (head === 'stay') {
      const stayLabels: Record<string, string> = {
        checkInDate: t('registration.stay.checkIn'),
        checkOutDate: t('registration.stay.checkOut'),
        estimatedArrivalTime: t('registration.stay.arrival'),
        plannedGuestCount: t('registration.stay.plannedCount'),
        bookingSource: t('registration.stay.bookingSource'),
        bookingReference: t('registration.stay.bookingReference')
      };
      return stayLabels[second] ?? second;
    }
    if (head === 'representative') return `${t('registration.steps.representative')}: ${guestLabel(second)}`;
    if (head === 'companions' && third) {
      return `${t('registration.companions.cardTitle', { index: Number(second) + 1 })}: ${guestLabel(third)}`;
    }
    if (head === 'companions') return t('registration.steps.companions');
    if (head === 'termsAccepted') return t('registration.review.termsLabel', { version: facility.terms_version });
    return t('registration.review.privacyLabel', { version: facility.privacy_version });
  };

  const showErrors = (errors: Record<string, string>) => {
    form.setErrors(errors);
    setSummary((prev) => ({
      focusKey: (prev?.focusKey ?? 0) + 1,
      items: Object.entries(errors).map(([path, message]) => ({
        fieldId: fieldId(path),
        label: labelForPath(path),
        message
      }))
    }));
  };

  const translateAll = (errors: FormErrorMap) =>
    Object.fromEntries(Object.entries(errors).map(([path, key]) => [path, translate(key)]));

  const goTo = (next: number) => {
    store.setStep(next);
    setSummary(null);
    mutation.reset();
    window.scrollTo({ top: 0 });
    requestAnimationFrame(() => headingRef.current?.focus());
  };

  const handleNext = () => {
    const errors = errorsForStep(validateWizard(form.getValues(), facility.max_guests), step);
    if (Object.keys(errors).length > 0) {
      showErrors(translateAll(errors));
      return;
    }
    form.clearErrors();
    goTo(step + 1);
  };

  const handleSubmit = () => {
    const current = form.getValues();
    const errors = validateWizard(current, facility.max_guests);
    if (Object.keys(errors).length > 0) {
      const firstStep = Math.min(...Object.keys(errors).map(stepOfPath));
      if (firstStep !== step) store.setStep(firstStep);
      showErrors(translateAll(firstStep === 3 ? errors : errorsForStep(errors, firstStep)));
      return;
    }
    setSummary(null);
    const idempotencyKey = store.ensureIdempotencyKey();
    mutation.mutate(
      { payload: buildCreatePayload(current, facility), idempotencyKey },
      {
        onSuccess: (result) => {
          if (!result.edit_token) {
            // Replay that cannot mint another link: same key kept, guest contacts the host.
            store.markAlreadySubmitted();
            return;
          }
          store.complete(result, facility);
          router.push(paths.registration.complete.getHref());
        },
        onError: (error) => {
          if (error instanceof ApiError && error.code === 'idempotency_conflict') {
            store.markAlreadySubmitted();
            return;
          }
          if (error instanceof ApiError && error.code === 'validation_error') {
            const mapped: Record<string, string> = {};
            for (const detail of error.details) {
              const path = wizardPathFromLocation(detail.location);
              if (path && !(path in mapped)) mapped[path] = t('validation.fieldInvalid');
            }
            if (Object.keys(mapped).length > 0) {
              const firstStep = Math.min(...Object.keys(mapped).map(stepOfPath));
              store.setStep(firstStep);
              showErrors(mapped);
            }
          }
        }
      }
    );
  };

  const submitError =
    mutation.isError && !(mutation.error instanceof ApiError && mutation.error.code === 'idempotency_conflict') ? (
      <ApiErrorBanner
        error={mutation.error}
        message={errorMessageKey(mutation.error) === 'network_error' ? t('registration.networkError') : undefined}
      />
    ) : null;

  return (
    <Stack gap='lg'>
      <FormStepper
        labels={stepLabels}
        active={step}
        ariaLabel={t('registration.progressLabel')}
        stepOfLabel={`${step + 1} / ${WIZARD_STEPS.length}`}
      />
      <form
        noValidate
        onSubmit={(event) => {
          event.preventDefault();
          if (step < 3) handleNext();
          else handleSubmit();
        }}
      >
        <Stack gap='lg'>
          <Title order={2} fz='xl' ref={headingRef} tabIndex={-1} style={{ outline: 'none' }}>
            {stepLabels[step]}
          </Title>
          {summary ? (
            <ValidationSummary
              title={t('validation.summaryTitle')}
              description={t('validation.summaryCount', { count: summary.items.length })}
              items={summary.items}
              focusKey={summary.focusKey}
            />
          ) : null}
          {step === 0 ? <StayStep form={form} facility={facility} /> : null}
          {step === 1 ? <RepresentativeStep form={form} representative={values.representative} /> : null}
          {step === 2 ? (
            <>
              <PartialWarning registered={values.companions.length + 1} planned={planned} />
              <CompanionsStep
                form={form}
                companions={values.companions}
                representative={values.representative}
                maxCompanions={maxCompanions}
                onAdd={() => form.insertListItem('companions', emptyGuest())}
                onRemove={(index) => form.removeListItem('companions', index)}
                companionsError={typeof form.errors.companions === 'string' ? form.errors.companions : undefined}
              />
            </>
          ) : null}
          {step === 3 ? <ReviewConsentStep form={form} values={values} facility={facility} onEditStep={goTo} /> : null}
          {submitError}
          <Group justify='space-between' gap='sm' wrap='wrap-reverse'>
            {step > 0 ? (
              <Button variant='default' onClick={() => goTo(step - 1)} disabled={mutation.isPending}>
                {t('registration.back')}
              </Button>
            ) : (
              <span />
            )}
            <Button type='submit' loading={mutation.isPending} style={{ flexGrow: 1, maxWidth: 360 }}>
              {step < 3 ? t('registration.next') : t('registration.submit')}
            </Button>
          </Group>
        </Stack>
      </form>
    </Stack>
  );
}

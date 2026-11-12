'use client';

import {
  Badge,
  Button,
  Card,
  Checkbox,
  Group,
  NativeSelect,
  Radio,
  SimpleGrid,
  Stack,
  Text,
  Textarea
} from '@mantine/core';
import { useLocale, useTranslations } from 'next-intl';
import { type FormEvent, useRef, useState } from 'react';
import { ButtonLink } from '@/components/ButtonLink';
import { InfoBanner } from '@/components/InfoBanner';
import { paths } from '@/config/paths';
import type { SurveyDefinition, SurveyQuestion } from '@/config/survey-definition';
import { useSubmitSurvey } from '@/features/survey/api/submit-survey';
import {
  buildSurveyPayload,
  type SurveyAnswers,
  type SurveyAnswerValue,
  surveyErrorKey,
  surveyErrors
} from '@/features/survey/utils/build-payload';
import type { Locale } from '@/i18n/routing';
import { ApiError } from '@/lib/api-client';

type Props = {
  definition: SurveyDefinition;
  facilities: { code: string; name: string }[];
  initialFacility: string;
};

const RATINGS = ['1', '2', '3', '4', '5'];

/**
 * Anonymous post-stay survey (DESIGN §5.5): large tap targets for ratings and choices, optional free
 * text, a hidden honeypot and a completion state. Nothing identifying is sent or stored locally.
 */
export function SurveyForm({ definition, facilities, initialFacility }: Props) {
  const t = useTranslations();
  const locale = useLocale() as Locale;
  const [facility, setFacility] = useState(initialFacility);
  const [answers, setAnswers] = useState<SurveyAnswers>({});
  const [showErrors, setShowErrors] = useState(false);
  const honeypot = useRef<HTMLInputElement>(null);
  const submit = useSubmitSurvey();
  const errors = surveyErrors(definition, answers);

  const set = (code: string, value: SurveyAnswerValue) => setAnswers((a) => ({ ...a, [code]: value }));

  const onSubmit = (e: FormEvent) => {
    e.preventDefault();
    if (Object.keys(errors).length > 0) {
      setShowErrors(true);
      document.getElementById(`survey-${Object.keys(errors)[0]}`)?.focus();
      return;
    }
    submit.mutate(buildSurveyPayload(definition, facility, answers, honeypot.current?.value ?? ''));
  };

  if (submit.isSuccess) {
    return (
      <Stack gap='md'>
        <InfoBanner tone='success' title={t('survey.thanksTitle')} announce='polite'>
          <Text>{t('survey.thanksBody')}</Text>
        </InfoBanner>
        <div>
          <ButtonLink href={paths.public.home.getHref()} variant='outline'>
            {t('survey.backHome')}
          </ButtonLink>
        </div>
      </Stack>
    );
  }

  const errorText = (code: string) => {
    if (!showErrors || !errors[code]) return undefined;
    return errors[code] === 'required' ? t('validation.required') : t('validation.tooLong');
  };

  const legend = (q: SurveyQuestion, n: number) => (
    <Group gap={8} wrap='nowrap' align='flex-start'>
      <Text component='span' fw={700} fz='md'>
        {n}. {q.label[locale]}
      </Text>
      {q.required ? (
        <Badge color='terracotta' variant='light' style={{ flexShrink: 0, textTransform: 'none' }}>
          {t('survey.required')}
        </Badge>
      ) : null}
    </Group>
  );

  const renderQuestion = (q: SurveyQuestion, n: number) => {
    const id = `survey-${q.code}`;
    const value = answers[q.code];
    switch (q.type) {
      case 'rating_1_5':
        return (
          <Radio.Group
            id={id}
            label={legend(q, n)}
            description={`${t('survey.ratingLow')} / ${t('survey.ratingHigh')}`}
            value={value === undefined ? null : String(value)}
            onChange={(v) => set(q.code, Number(v))}
            error={errorText(q.code)}
          >
            <SimpleGrid cols={5} spacing={8} mt='sm'>
              {RATINGS.map((r) => (
                <Radio.Card
                  key={r}
                  value={r}
                  radius='md'
                  aria-label={t('survey.ratingOption', { value: r })}
                  mih={48}
                  style={{ display: 'grid', placeItems: 'center' }}
                >
                  <Text fw={700} fz='lg'>
                    {r}
                  </Text>
                </Radio.Card>
              ))}
            </SimpleGrid>
          </Radio.Group>
        );
      case 'single_choice':
        return (
          <Radio.Group
            id={id}
            label={legend(q, n)}
            value={typeof value === 'string' ? value : null}
            onChange={(v) => set(q.code, v)}
            error={errorText(q.code)}
          >
            <Stack gap={4} mt='sm'>
              {q.options.map((o) => (
                <Radio key={o.value} value={o.value} label={o.label[locale]} size='md' py={10} />
              ))}
            </Stack>
          </Radio.Group>
        );
      case 'multi_choice':
        return (
          <Checkbox.Group
            id={id}
            label={legend(q, n)}
            value={Array.isArray(value) ? value : []}
            onChange={(v) => set(q.code, v)}
            error={errorText(q.code)}
          >
            <Stack gap={4} mt='sm'>
              {q.options.map((o) => (
                <Checkbox key={o.value} value={o.value} label={o.label[locale]} size='md' py={10} />
              ))}
            </Stack>
          </Checkbox.Group>
        );
      case 'text': {
        const text = typeof value === 'string' ? value : '';
        return (
          <Textarea
            id={id}
            label={legend(q, n)}
            description={t('survey.textHelp')}
            value={text}
            onChange={(e) => set(q.code, e.currentTarget.value)}
            autosize
            minRows={4}
            maxLength={q.maxLength}
            autoComplete='off'
            error={errorText(q.code)}
          />
        );
      }
    }
  };

  return (
    <form onSubmit={onSubmit} noValidate>
      <Stack gap='lg'>
        {facilities.length > 1 ? (
          <NativeSelect
            label={t('survey.facility')}
            value={facility}
            onChange={(e) => setFacility(e.currentTarget.value)}
            data={facilities.map((f) => ({ value: f.code, label: f.name }))}
          />
        ) : null}
        {definition.questions.map((q, i) => (
          <Card key={q.code} padding='lg'>
            {renderQuestion(q, i + 1)}
            {q.type === 'text' ? (
              <Text size='xs' c='dimmed' ta='right' mt={4}>
                {t('survey.charCount', {
                  count: [...(typeof answers[q.code] === 'string' ? (answers[q.code] as string) : '')].length,
                  max: q.maxLength
                })}
              </Text>
            ) : null}
          </Card>
        ))}
        {/* Honeypot: hidden from people and assistive tech; bots that fill it are discarded server-side. */}
        <div
          aria-hidden='true'
          style={{ position: 'absolute', left: '-10000px', width: 1, height: 1, overflow: 'hidden' }}
        >
          <label>
            {t('survey.honeypot')}
            <input ref={honeypot} type='text' name='website' tabIndex={-1} autoComplete='off' defaultValue='' />
          </label>
        </div>
        {submit.isError ? (
          <InfoBanner tone='error' title={t('survey.errorTitle')} announce='assertive'>
            <Text size='sm'>
              {t(
                `errors.${surveyErrorKey(
                  submit.error instanceof ApiError ? submit.error.code : undefined,
                  submit.error instanceof ApiError ? submit.error.status : 0
                )}`
              )}
            </Text>
          </InfoBanner>
        ) : null}
        <Button type='submit' size='lg' loading={submit.isPending} fullWidth maw={{ sm: 320 }}>
          {t('survey.submit')}
        </Button>
      </Stack>
    </form>
  );
}

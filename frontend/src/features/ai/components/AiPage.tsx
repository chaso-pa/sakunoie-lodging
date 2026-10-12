'use client';

import { Button, NativeSelect, Textarea, UnstyledButton } from '@mantine/core';
import { useLocale, useTranslations } from 'next-intl';
import { useState } from 'react';
import { EmptyState, ErrorState, LoadingState } from '@/components/QueryState';
import { paths } from '@/config/paths';
import { aiMessageMaxLength, useAskAi } from '@/features/ai/api/ask-ai';
import {
  type AiFilters,
  aiDurations,
  aiGenres,
  aiPurposes,
  aiTransports,
  aiWeathers,
  useAiSpots
} from '@/features/ai/api/get-ai-spots';
import classes from '@/features/ai/components/Ai.module.scss';
import { isLocale, type Locale, localeLabels, locales } from '@/i18n/config';
import { getApiErrorStatus } from '@/lib/api-client';

// Message key of each stored genre. Kept in list order because "・" is not safe as an unquoted object key.
const genreKeys = ['hidden', 'nature', 'history', 'cafe'] as const;

/** Narrows a <select> value to one of its own options. */
const pick = <T extends string>(options: readonly T[], value: string, fallback: T): T =>
  options.find((option) => option === value) ?? fallback;

/** AI spot assistant: curated suggestions by condition, and a free-form question to the AI. */
export function AiPage() {
  const t = useTranslations('community.ai');
  const locale = useLocale();
  const askAi = useAskAi();

  const [filters, setFilters] = useState<AiFilters>({
    genre: aiGenres[0],
    purpose: 'all',
    weather: 'any',
    duration: 'half',
    transport: 'car'
  });
  const [answerLanguage, setAnswerLanguage] = useState<Locale>(locale);
  // The conditions the listed suggestions were asked with; null until the member asks.
  const [applied, setApplied] = useState<AiFilters | null>(null);
  const [message, setMessage] = useState('');
  const [inputError, setInputError] = useState('');

  const suggestions = useAiSpots(applied);
  const update = (patch: Partial<AiFilters>) => setFilters((current) => ({ ...current, ...patch }));

  const ask = () => {
    const text = message.trim();
    if (!text) {
      setInputError(t('chat.empty'));
      return;
    }
    setInputError('');
    askAi.mutate({ ...filters, message: text, language: answerLanguage });
  };

  const chatStatus = () => {
    if (inputError) {
      return { text: inputError, isError: true };
    }
    if (askAi.isPending) {
      return { text: t('chat.asking'), isError: false };
    }
    if (askAi.isError) {
      const status = getApiErrorStatus(askAi.error);
      const text =
        status === 503
          ? t('chat.errors.notConfigured')
          : status === 502
            ? t('chat.errors.unavailable')
            : status === 429
              ? t('chat.errors.rateLimited')
              : status === 422 || status === 400
                ? t('chat.errors.invalid')
                : status === undefined
                  ? t('chat.errors.network')
                  : t('chat.errors.generic');
      return { text, isError: true };
    }
    if (askAi.isSuccess) {
      return { text: t('chat.answered'), isError: false };
    }
    return { text: '', isError: false };
  };
  const status = chatStatus();

  return (
    <section className={classes.feature}>
      <div className={classes.head}>
        <div className={classes.icon} aria-hidden>
          ✦
        </div>
        <div>
          <h1>{t('title')}</h1>
          <div className={classes.sub}>{t('sub')}</div>
        </div>
      </div>
      <p className={classes.lead}>{t('lead')}</p>

      {/* biome-ignore lint/a11y/useSemanticElements: a wrapping row of toggle buttons, not a form fieldset */}
      <div className={classes.quick} role='group' aria-label={t('genreLabel')}>
        {aiGenres.map((genre, index) => (
          <UnstyledButton
            key={genre}
            className={classes.quickBtn}
            aria-pressed={filters.genre === genre}
            onClick={() => update({ genre })}
          >
            {t(`genres.${genreKeys[index]}`)}
          </UnstyledButton>
        ))}
      </div>
      <div className={classes.groupLabel} id='ai-purpose-label'>
        {t('purposeLabel')}
      </div>
      {/* biome-ignore lint/a11y/useSemanticElements: a wrapping row of toggle buttons, not a form fieldset */}
      <div className={classes.quick} role='group' aria-labelledby='ai-purpose-label'>
        {aiPurposes.map((purpose) => (
          <UnstyledButton
            key={purpose}
            className={classes.quickBtn}
            aria-pressed={filters.purpose === purpose}
            onClick={() => update({ purpose })}
          >
            {t(`purposes.${purpose}`)}
          </UnstyledButton>
        ))}
      </div>
      <div className={classes.options}>
        <NativeSelect
          size='sm'
          radius='sm'
          aria-label={t('weatherLabel')}
          value={filters.weather}
          onChange={(event) => update({ weather: pick(aiWeathers, event.currentTarget.value, 'any') })}
          data={aiWeathers.map((value) => ({ value, label: t(`weathers.${value}`) }))}
        />
        <NativeSelect
          size='sm'
          radius='sm'
          aria-label={t('durationLabel')}
          value={filters.duration}
          onChange={(event) => update({ duration: pick(aiDurations, event.currentTarget.value, 'half') })}
          data={aiDurations.map((value) => ({ value, label: t(`durations.${value}`) }))}
        />
        <NativeSelect
          size='sm'
          radius='sm'
          aria-label={t('transportLabel')}
          value={filters.transport}
          onChange={(event) => update({ transport: pick(aiTransports, event.currentTarget.value, 'car') })}
          data={aiTransports.map((value) => ({ value, label: t(`transports.${value}`) }))}
        />
        <NativeSelect
          size='sm'
          radius='sm'
          aria-label={t('answerLanguageLabel')}
          value={answerLanguage}
          onChange={(event) => {
            const next = event.currentTarget.value;
            if (isLocale(next)) {
              setAnswerLanguage(next);
            }
          }}
          data={locales.map((value) => ({ value, label: localeLabels[value] }))}
        />
      </div>
      <Button fullWidth color='#3f705d' onClick={() => setApplied({ ...filters })}>
        ✦ {t('showSuggestions')}
      </Button>

      <div className={classes.chatBox}>
        <h2 className={classes.chatTitle}>✦ {t('chat.title')}</h2>
        <div className={classes.chatLead}>{t('chat.lead')}</div>
        <Textarea
          aria-label={t('chat.inputLabel')}
          placeholder={t('chat.placeholder')}
          size='md'
          radius='md'
          minRows={3}
          autosize
          maxLength={aiMessageMaxLength}
          value={message}
          onChange={(event) => setMessage(event.currentTarget.value)}
        />
        <Button fullWidth mt={8} color='#3f705d' loading={askAi.isPending} onClick={ask}>
          {t('chat.submit')}
        </Button>
        <div className={classes.status} data-error={status.isError || undefined} aria-live='polite'>
          {status.text}
        </div>
        {/* The answer is model output: React renders it as text, so any markup in it stays inert. */}
        {askAi.isSuccess && !askAi.isPending && (
          <div className={classes.answer}>{askAi.data.answer || t('chat.emptyAnswer')}</div>
        )}
        <div className={classes.disclaimer}>{t('chat.disclaimer')}</div>
      </div>

      {applied && (
        <div className={classes.results} aria-live='polite'>
          {suggestions.isPending ? (
            <LoadingState />
          ) : suggestions.isError ? (
            <ErrorState
              error={suggestions.error}
              onRetry={() => suggestions.refetch()}
              retrying={suggestions.isRefetching}
            />
          ) : suggestions.data.length === 0 ? (
            <EmptyState>{t('noSuggestions')}</EmptyState>
          ) : (
            suggestions.data.map((spot, index) => (
              <div key={spot.id} className={classes.result}>
                <span className={classes.communityTag}>
                  {t('suggestionTag', { rank: index + 1 })} · {spot.kind}
                </span>
                <strong>{spot.name}</strong>
                <p>{spot.description}</p>
                <div className={classes.meta}>
                  📍 {spot.area} · {applied.transport === 'car' ? '🚗' : '🚃'}{' '}
                  {t('travelTime', { minutes: applied.transport === 'car' ? spot.drive_min : spot.transit_min })} · ⏱{' '}
                  {t('stayTime', { minutes: spot.stay_min })}
                </div>
                <div className={classes.links}>
                  <a
                    href={paths.external.googleMapsSearch.getHref(spot.map_query)}
                    target='_blank'
                    rel='noopener noreferrer'
                  >
                    📍 {t('mapLink')}
                  </a>
                  <a href={spot.search_url} target='_blank' rel='noopener noreferrer'>
                    {t('detailLink')} ↗
                  </a>
                </div>
              </div>
            ))
          )}
          {suggestions.isSuccess && suggestions.data.length > 0 && (
            <div className={classes.disclaimer}>{t('suggestionNote')}</div>
          )}
        </div>
      )}
    </section>
  );
}

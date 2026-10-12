'use client';

import { Button, Rating, Textarea } from '@mantine/core';
import { notifications } from '@mantine/notifications';
import { useTranslations } from 'next-intl';
import { useState } from 'react';
import shared from '@/components/Community.module.scss';
import { SampleBadge } from '@/components/SampleBadge';
import { TranslatableText } from '@/components/TranslatableText';
import { reviewTextMaxLength, useCreateReview } from '@/features/map/api/create-review';
import classes from '@/features/map/components/Map.module.scss';
import { getApiErrorStatus } from '@/lib/api-client';
import type { Review } from '@/types/community';

export function ReviewSection({ spotId, reviews }: { spotId: string; reviews: Review[] }) {
  const t = useTranslations('community.spot');
  const tErrors = useTranslations('errors');
  const createReview = useCreateReview();
  const [stars, setStars] = useState(5);
  const [text, setText] = useState('');
  const [error, setError] = useState('');

  const submit = (event: React.FormEvent) => {
    event.preventDefault();
    if (!text.trim()) {
      setError(t('textRequired'));
      return;
    }
    setError('');
    createReview.mutate(
      { spotId, stars, text: text.trim() },
      {
        onSuccess: () => {
          setText('');
          setStars(5);
          notifications.show({ message: t('added') });
        },
        onError: (failure) => {
          const status = getApiErrorStatus(failure);
          setError(
            status === undefined ? tErrors('network') : status === 429 ? tErrors('rateLimited') : tErrors('generic')
          );
        }
      }
    );
  };

  return (
    <>
      <h2 className={shared.sectionTitle}>{t('reviewsTitle')}</h2>
      {reviews.length === 0 ? (
        <p className={shared.muted}>{t('noReviews')}</p>
      ) : (
        reviews.map((review) => (
          <div key={review.id} className={classes.review}>
            <strong>{review.member.nickname}</strong>{' '}
            <span className={classes.stars} role='img' aria-label={t('starsValue', { stars: review.stars })}>
              {'★'.repeat(Math.max(0, Math.min(5, review.stars)))}
            </span>{' '}
            {review.is_sample && <SampleBadge />}
            <TranslatableText text={review.text} />
          </div>
        ))
      )}

      <form onSubmit={submit} noValidate>
        <h3 className={shared.h3} style={{ marginTop: 20 }}>
          {t('writeTitle')}
        </h3>
        <div className={shared.fieldLabel} id='review-stars-label'>
          {t('starsLabel')}
        </div>
        <Rating
          value={stars}
          onChange={(value) => setStars(Math.max(1, value))}
          size='lg'
          aria-labelledby='review-stars-label'
          getSymbolLabel={(value) => t('starsValue', { stars: value })}
        />
        <Textarea
          className={shared.field}
          label={t('textLabel')}
          placeholder={t('textPlaceholder')}
          size='md'
          radius='sm'
          classNames={{ label: shared.fieldLabel }}
          minRows={3}
          autosize
          maxLength={reviewTextMaxLength}
          value={text}
          onChange={(event) => setText(event.currentTarget.value)}
        />
        <p className={shared.formError} role='alert'>
          {error}
        </p>
        <Button type='submit' fullWidth size='md' loading={createReview.isPending}>
          {t('submit')}
        </Button>
      </form>
    </>
  );
}

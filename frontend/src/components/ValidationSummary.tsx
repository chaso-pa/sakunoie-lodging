'use client';

import { Anchor, List, Text } from '@mantine/core';
import { useEffect, useRef } from 'react';
import { InfoBanner } from '@/components/InfoBanner';

export type ValidationSummaryItem = {
  /** DOM id of the field, used as an in-page link target. */
  fieldId?: string;
  label: string;
  message: string;
};

type ValidationSummaryProps = {
  title: string;
  description?: string;
  items: ValidationSummaryItem[];
  /** Changes on every submit so focus moves to the summary again. */
  focusKey?: number;
};

/** Error summary announced with role="alert" and focused after a failed submit. */
export function ValidationSummary({ title, description, items, focusKey }: ValidationSummaryProps) {
  const ref = useRef<HTMLDivElement>(null);

  // biome-ignore lint/correctness/useExhaustiveDependencies: focusKey is the trigger by design
  useEffect(() => {
    ref.current?.focus();
  }, [focusKey]);

  return (
    <div ref={ref} tabIndex={-1} style={{ outline: 'none' }}>
      <InfoBanner tone='error' title={title} announce='assertive'>
        {description ? <Text size='sm'>{description}</Text> : null}
        {items.length > 0 ? (
          <List size='sm' mt={4}>
            {items.map((item) => (
              <List.Item key={`${item.fieldId ?? item.label}-${item.message}`}>
                {item.fieldId ? (
                  <Anchor href={`#${item.fieldId}`} size='sm'>
                    {item.label}
                  </Anchor>
                ) : (
                  item.label
                )}
                {`: ${item.message}`}
              </List.Item>
            ))}
          </List>
        ) : null}
      </InfoBanner>
    </div>
  );
}

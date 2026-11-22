import { Badge, Group, Title } from '@mantine/core';
import type { ReactNode } from 'react';

type Props = {
  id: string;
  title: string;
  /** Visible "placeholder" marker next to the heading for content that is not confirmed yet. */
  placeholderLabel?: string;
  children: ReactNode;
};

/** One anchored page section with a real h2 (the in-page nav links to `#id`). */
export function FacilitySection({ id, title, placeholderLabel, children }: Props) {
  return (
    <section id={id} aria-labelledby={`${id}-title`} style={{ scrollMarginTop: 24 }}>
      <Group gap='sm' mb={{ base: 'sm', sm: 'lg' }} align='center'>
        <Title id={`${id}-title`} order={2} fz={{ base: 22, md: 26 }} lh={1.3}>
          {title}
        </Title>
        {placeholderLabel ? <PlaceholderMark label={placeholderLabel} /> : null}
      </Group>
      {children}
    </section>
  );
}

/** Small "placeholder" tag: text plus an outline, never colour alone. */
export function PlaceholderMark({ label }: { label: string }) {
  return (
    <Badge
      variant='outline'
      radius='sm'
      size='md'
      style={{
        textTransform: 'none',
        color: 'var(--sakunoie-warning-text)',
        borderColor: 'var(--sakunoie-warning-text)',
        background: 'var(--sakunoie-surface)'
      }}
    >
      {label}
    </Badge>
  );
}

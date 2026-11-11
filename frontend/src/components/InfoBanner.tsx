import { Group, Paper, Stack, Text } from '@mantine/core';
import { IconAlertTriangle, IconCircleCheck, IconCircleX, IconInfoCircle } from '@tabler/icons-react';
import type { ReactNode } from 'react';

type Tone = 'info' | 'success' | 'warning' | 'error';

const toneStyle: Record<Tone, { color: string; Icon: typeof IconInfoCircle }> = {
  info: { color: 'var(--sakunoie-forest)', Icon: IconInfoCircle },
  success: { color: 'var(--sakunoie-success)', Icon: IconCircleCheck },
  // AA-safe warning text shade (see config/theme.ts).
  warning: { color: 'var(--sakunoie-warning-text)', Icon: IconAlertTriangle },
  error: { color: 'var(--sakunoie-error)', Icon: IconCircleX }
};

type InfoBannerProps = {
  tone?: Tone;
  title?: ReactNode;
  children?: ReactNode;
  /** Live-region behavior: `polite` (role=status) or `assertive` (role=alert). */
  announce?: 'polite' | 'assertive';
};

/** Status message with text plus icon (never color alone, DESIGN §2.6). */
export function InfoBanner({ tone = 'info', title, children, announce }: InfoBannerProps) {
  const { color, Icon } = toneStyle[tone];
  const role = announce === 'assertive' ? 'alert' : announce === 'polite' ? 'status' : undefined;
  return (
    <Paper
      role={role}
      p='md'
      radius='md'
      bg='var(--sakunoie-surface-muted)'
      style={{ borderLeft: `4px solid ${color}` }}
    >
      <Group align='flex-start' gap='sm' wrap='nowrap'>
        <Icon size={22} color={color} aria-hidden style={{ flexShrink: 0, marginTop: 2 }} />
        <Stack gap={4}>
          {title ? (
            <Text fw={700} c={color}>
              {title}
            </Text>
          ) : null}
          {children ? <div>{children}</div> : null}
        </Stack>
      </Group>
    </Paper>
  );
}

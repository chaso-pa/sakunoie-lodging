import { Paper, Text } from '@mantine/core';
import type { ReactNode } from 'react';

export function MetricCard({ label, value, icon }: { label: string; value: number | string; icon?: ReactNode }) {
  return (
    <Paper withBorder radius='lg' p='md' bg='var(--sakunoie-surface)'>
      <Text size='sm' c='dimmed' fw={600} style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
        {icon}
        {label}
      </Text>
      <Text fz={28} fw={700} lh={1.2} mt={4}>
        {value}
      </Text>
    </Paper>
  );
}

import { Group, Paper, Text } from '@mantine/core';
import { FeatureIcon } from '@/features/public-site/components/FeatureCard';

/** Static amenity label (not an interactive chip). */
export function AmenityChip({ icon, label }: { icon: string; label: string }) {
  return (
    <Paper bg='var(--sakunoie-surface-muted)' radius='md' px='sm' py={10} mih={44}>
      <Group gap={8} wrap='nowrap'>
        <FeatureIcon name={icon} size={20} />
        <Text size='sm' fw={500}>
          {label}
        </Text>
      </Group>
    </Paper>
  );
}

import { Group, Text } from '@mantine/core';
import { IconAlertTriangle, IconCircleCheck } from '@tabler/icons-react';
import { useTranslations } from 'next-intl';

type RegistrationStatusProps = { registered: number; planned: number; remaining: number; isComplete: boolean };

/** "2/3 registered · 1 remaining" in text plus icon, never color alone. */
export function RegistrationStatus({ registered, planned, remaining, isComplete }: RegistrationStatusProps) {
  const t = useTranslations('status');
  const color = isComplete ? 'var(--sakunoie-success)' : 'var(--sakunoie-warning-text)';
  const Icon = isComplete ? IconCircleCheck : IconAlertTriangle;
  return (
    <Group gap='xs' wrap='nowrap' align='flex-start'>
      <Icon size={22} color={color} aria-hidden style={{ flexShrink: 0 }} />
      <div>
        <Text fw={700} c={color}>
          {t('registeredOf', { registered, planned })}
          {isComplete ? null : ` · ${t('remaining', { remaining })}`}
        </Text>
        <Text size='sm' c='dimmed'>
          {isComplete ? t('complete') : t('incomplete')}
        </Text>
      </div>
    </Group>
  );
}

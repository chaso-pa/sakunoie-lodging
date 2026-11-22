import { Group, Text } from '@mantine/core';
import {
  IconAlertTriangle,
  IconBan,
  IconCircleCheck,
  IconClockHour4,
  IconHourglassEmpty,
  IconLink,
  IconShieldCheck
} from '@tabler/icons-react';
import type { InvitationStatus } from '@/features/admin/types/api';

type Kind = 'complete' | 'incomplete' | 'verified' | 'pending' | 'linkActive' | 'linkExpired' | 'linkRevoked';

const style: Record<Kind, { color: string; Icon: typeof IconCircleCheck }> = {
  complete: { color: 'var(--sakunoie-success)', Icon: IconCircleCheck },
  incomplete: { color: 'var(--sakunoie-warning-text)', Icon: IconAlertTriangle },
  verified: { color: 'var(--sakunoie-forest)', Icon: IconShieldCheck },
  pending: { color: 'var(--mantine-color-dimmed)', Icon: IconClockHour4 },
  linkActive: { color: 'var(--sakunoie-forest)', Icon: IconLink },
  linkExpired: { color: 'var(--mantine-color-dimmed)', Icon: IconHourglassEmpty },
  linkRevoked: { color: 'var(--sakunoie-warning-text)', Icon: IconBan }
};

/** Status with text plus icon, never color alone (DESIGN §2.6, §5.4). */
export function StatusChip({ kind, label }: { kind: Kind; label: string }) {
  const { color, Icon } = style[kind];
  return (
    <Group
      gap={4}
      wrap='nowrap'
      px={8}
      py={2}
      style={{ border: `1px solid ${color}`, borderRadius: 999, display: 'inline-flex', background: '#fff' }}
    >
      <Icon size={16} color={color} aria-hidden />
      <Text size='sm' fw={600} c={color} style={{ whiteSpace: 'nowrap' }}>
        {label}
      </Text>
    </Group>
  );
}

export const completenessChip = (registered: number, planned: number, remaining: number) =>
  registered >= planned ? (
    <StatusChip kind='complete' label={`登録完了 ${registered}/${planned}名`} />
  ) : (
    <StatusChip kind='incomplete' label={`未登録あり 残り${remaining}名`} />
  );

export const identityChip = (verified: boolean) =>
  verified ? <StatusChip kind='verified' label='確認済み' /> : <StatusChip kind='pending' label='本人確認待ち' />;

export const invitationStatusLabel: Record<InvitationStatus, string> = {
  active: '有効',
  used: '登録済み',
  expired: '期限切れ',
  revoked: '取り消し済み'
};

const invitationKind: Record<InvitationStatus, Kind> = {
  active: 'linkActive',
  used: 'complete',
  expired: 'linkExpired',
  revoked: 'linkRevoked'
};

export const invitationChip = (status: InvitationStatus) => (
  <StatusChip kind={invitationKind[status]} label={invitationStatusLabel[status]} />
);

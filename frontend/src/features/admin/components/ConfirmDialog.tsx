'use client';

import { Button, Group, Modal, Stack, Text } from '@mantine/core';
import type { ReactNode } from 'react';

type ConfirmDialogProps = {
  opened: boolean;
  title: string;
  children: ReactNode;
  confirmLabel: string;
  onConfirm: () => void;
  onClose: () => void;
  loading?: boolean;
  danger?: boolean;
};

/** Explicit confirmation before an admin action (DESIGN §8). */
export function ConfirmDialog({
  opened,
  title,
  children,
  confirmLabel,
  onConfirm,
  onClose,
  loading,
  danger
}: ConfirmDialogProps) {
  return (
    <Modal opened={opened} onClose={onClose} title={<Text fw={700}>{title}</Text>} centered>
      <Stack gap='lg'>
        <div>{children}</div>
        <Group justify='flex-end'>
          <Button variant='default' onClick={onClose} disabled={loading}>
            キャンセル
          </Button>
          <Button color={danger ? 'red' : 'forest'} onClick={onConfirm} loading={loading}>
            {confirmLabel}
          </Button>
        </Group>
      </Stack>
    </Modal>
  );
}

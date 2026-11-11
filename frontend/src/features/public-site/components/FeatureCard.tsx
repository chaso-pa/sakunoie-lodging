import { Card, Text, ThemeIcon, Title } from '@mantine/core';
import { IconDoor, IconHome, IconMapPin, IconUsers } from '@tabler/icons-react';

const icons = { door: IconDoor, users: IconUsers, map: IconMapPin, home: IconHome } as const;
export type FeatureIconName = keyof typeof icons;

export function FeatureIcon({ name, size = 22 }: { name: string; size?: number }) {
  const Icon = icons[name as FeatureIconName] ?? IconHome;
  return <Icon size={size} aria-hidden stroke={1.6} />;
}

export function FeatureCard({ icon, title, body }: { icon: string; title: string; body: string }) {
  return (
    <Card padding='lg' h='100%'>
      <ThemeIcon variant='light' color='forest' size={44} radius='md' mb='sm'>
        <FeatureIcon name={icon} />
      </ThemeIcon>
      <Title order={3} fz='lg' mb={6}>
        {title}
      </Title>
      <Text c='dimmed'>{body}</Text>
    </Card>
  );
}

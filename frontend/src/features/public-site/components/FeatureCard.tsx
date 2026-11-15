import { Card, Text, ThemeIcon, Title } from '@mantine/core';
import {
  IconAirConditioning,
  IconBath,
  IconBed,
  IconBottle,
  IconClockHour3,
  IconCoffee,
  IconDental,
  IconDeviceProjector,
  IconDoor,
  IconHome,
  IconMapPin,
  IconMoodKid,
  IconMoon,
  IconMusicOff,
  IconParking,
  IconRulerMeasure,
  IconSmokingNo,
  IconSofa,
  IconStack2,
  IconToiletPaper,
  IconUserCheck,
  IconUsers,
  IconWashMachine,
  IconWifi
} from '@tabler/icons-react';
import type { FeatureIconName } from '@/features/public-site/types/icons';

const icons: Record<FeatureIconName, typeof IconHome> = {
  door: IconDoor,
  users: IconUsers,
  map: IconMapPin,
  bed: IconBed,
  projector: IconDeviceProjector,
  sofa: IconSofa,
  bath: IconBath,
  coffee: IconCoffee,
  parking: IconParking,
  wifi: IconWifi,
  ruler: IconRulerMeasure,
  clock: IconClockHour3,
  toilet: IconToiletPaper,
  laundry: IconWashMachine,
  toothbrush: IconDental,
  skincare: IconBottle,
  // Tabler has no towel icon; a stack reads as folded towels.
  towel: IconStack2,
  airConditioning: IconAirConditioning,
  smokingNo: IconSmokingNo,
  moon: IconMoon,
  musicOff: IconMusicOff,
  userCheck: IconUserCheck,
  child: IconMoodKid
};

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

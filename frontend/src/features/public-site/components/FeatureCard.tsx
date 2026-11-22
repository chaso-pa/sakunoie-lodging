import {
  IconAirConditioning,
  IconBath,
  IconBed,
  IconBottle,
  IconBread,
  IconCar,
  IconClockHour3,
  IconCoffee,
  IconDental,
  IconDeviceProjector,
  IconDoor,
  IconHome,
  IconKey,
  IconMapPin,
  IconMoodKid,
  IconMoon,
  IconMountain,
  IconMusicOff,
  IconParking,
  IconRulerMeasure,
  IconShoppingCart,
  IconSmokingNo,
  IconSofa,
  IconStack2,
  IconToiletPaper,
  IconToolsKitchen2,
  IconTrain,
  IconUserCheck,
  IconUsers,
  IconWalk,
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
  child: IconMoodKid,
  car: IconCar,
  key: IconKey,
  home: IconHome,
  walk: IconWalk,
  train: IconTrain,
  mountain: IconMountain,
  restaurant: IconToolsKitchen2,
  bread: IconBread,
  cart: IconShoppingCart
};

/** Line icon for a content icon name (`types/icons.ts`); unknown names fall back to a house. */
export function FeatureIcon({ name, size = 22 }: { name: string; size?: number }) {
  const Icon = icons[name as FeatureIconName] ?? IconHome;
  return <Icon size={size} aria-hidden stroke={1.6} />;
}

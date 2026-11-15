/**
 * Icon names content may use (facility JSON `amenities[].icon`, front matter `icon` fields). Kept free
 * of React so the Zod content schemas and tests can import it; `FeatureIcon` maps each name to a
 * Tabler icon and the compiler checks the map is complete.
 */
export const featureIconNames = [
  'door',
  'users',
  'map',
  'bed',
  'projector',
  'sofa',
  'bath',
  'coffee',
  'parking',
  'wifi',
  'ruler',
  'clock',
  'toilet',
  'laundry',
  'toothbrush',
  'skincare',
  'towel',
  'airConditioning',
  'smokingNo',
  'moon',
  'musicOff',
  'userCheck',
  'child',
  'car',
  'key',
  'home',
  'walk',
  'train',
  'mountain',
  'restaurant',
  'bread',
  'cart'
] as const;

export type FeatureIconName = (typeof featureIconNames)[number];

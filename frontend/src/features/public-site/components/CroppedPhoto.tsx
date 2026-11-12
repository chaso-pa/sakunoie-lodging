import { AspectRatio, Badge, Box } from '@mantine/core';
import Image from 'next/image';
import type { FacilityPhoto } from '@/config/site';

type Props = {
  photo: FacilityPhoto;
  alt: string;
  badge: string;
  /** Display ratio; defaults to the asset's own ratio. */
  ratio?: number;
  sizes?: string;
  priority?: boolean;
  radius?: number;
};

/**
 * A facility photo cropped around its focal point inside a fixed-ratio box, so it causes no layout
 * shift. A photo flagged `placeholder` carries a visible badge. Files are served as-is (`unoptimized`).
 */
export function CroppedPhoto({ photo, alt, badge, ratio, sizes = '100vw', priority, radius = 14 }: Props) {
  const [w, h] = photo.aspectRatio.split(':').map(Number);
  return (
    <AspectRatio ratio={ratio ?? w / h} style={{ borderRadius: radius, overflow: 'hidden', position: 'relative' }}>
      <Box pos='relative' w='100%' h='100%'>
        <Image
          src={photo.src}
          alt={alt}
          fill
          sizes={sizes}
          priority={priority}
          unoptimized
          style={{ objectFit: 'cover', objectPosition: photo.focalPoint }}
        />
        {photo.placeholder ? (
          <Badge
            pos='absolute'
            top={8}
            left={8}
            color='dark'
            variant='filled'
            radius='sm'
            style={{ textTransform: 'none' }}
          >
            {badge}
          </Badge>
        ) : null}
      </Box>
    </AspectRatio>
  );
}

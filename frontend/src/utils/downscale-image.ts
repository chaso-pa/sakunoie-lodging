/** Longest side of an uploaded photo. The API resizes again; this keeps phone photos under its 10MB cap. */
export const uploadMaxSide = 1600;
/** The API rejects request bodies over 10MB. Leave room for the other form fields. */
export const uploadMaxBytes = 9.5 * 1024 * 1024;
/** Refuse to even decode files beyond this size. */
export const sourceMaxBytes = 40 * 1024 * 1024;

export type ImageProblem = 'type' | 'tooLarge' | 'unreadable';

export class ImageError extends Error {
  constructor(public readonly problem: ImageProblem) {
    super(problem);
  }
}

const loadImage = (file: File) =>
  new Promise<HTMLImageElement>((resolve, reject) => {
    const url = URL.createObjectURL(file);
    const image = new Image();
    image.onload = () => {
      URL.revokeObjectURL(url);
      resolve(image);
    };
    image.onerror = () => {
      URL.revokeObjectURL(url);
      reject(new ImageError('unreadable'));
    };
    image.src = url;
  });

/**
 * Re-encodes a chosen photo as a JPEG whose longest side is at most `uploadMaxSide`. Drawing to a canvas
 * also drops the metadata the file carried; the API strips EXIF again on its side regardless.
 * Throws ImageError when the file is not an image, cannot be decoded, or is still too large.
 */
export const downscaleImage = async (file: File): Promise<File> => {
  if (!file.type.startsWith('image/')) {
    throw new ImageError('type');
  }
  if (file.size > sourceMaxBytes) {
    throw new ImageError('tooLarge');
  }

  const image = await loadImage(file);
  const scale = Math.min(1, uploadMaxSide / Math.max(image.naturalWidth, image.naturalHeight));
  const canvas = document.createElement('canvas');
  canvas.width = Math.max(1, Math.round(image.naturalWidth * scale));
  canvas.height = Math.max(1, Math.round(image.naturalHeight * scale));
  const context = canvas.getContext('2d');
  if (!context) {
    throw new ImageError('unreadable');
  }
  // JPEG has no alpha channel: paint transparent PNG/WebP areas white instead of black.
  context.fillStyle = '#fff';
  context.fillRect(0, 0, canvas.width, canvas.height);
  context.drawImage(image, 0, 0, canvas.width, canvas.height);

  const blob = await new Promise<Blob | null>((resolve) => canvas.toBlob(resolve, 'image/jpeg', 0.82));
  if (!blob) {
    throw new ImageError('unreadable');
  }
  if (blob.size > uploadMaxBytes) {
    throw new ImageError('tooLarge');
  }
  return new File([blob], 'photo.jpg', { type: 'image/jpeg' });
};

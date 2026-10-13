import axios, { isAxiosError } from 'axios';
import { env } from '@/config/env';

/**
 * Shared axios instance. Requests go to the same-origin `/api` path, which Next.js proxies to the Go API,
 * so the session cookie is sent without any CORS setup.
 */
export const api = axios.create({
  baseURL: `${env.apiBaseUrl}/api`,
  withCredentials: true
});

/** HTTP status of a failed API call, or undefined when the request never got a response. */
export const getApiErrorStatus = (error: unknown): number | undefined =>
  isAxiosError(error) ? error.response?.status : undefined;

const isRecord = (value: unknown): value is Record<string, unknown> => typeof value === 'object' && value !== null;

/**
 * True when an upload was refused for its size. The Go API caps request bodies at 10MB and answers 422
 * with "request body too large"; above 12MB the Next.js proxy gives up first with a bare 500 that carries
 * no API error body. 413 is included for any proxy placed in front later.
 */
export const isUploadTooLarge = (error: unknown): boolean => {
  if (!isAxiosError(error) || !error.response) {
    return false;
  }
  const { status, data } = error.response;
  if (status === 413) {
    return true;
  }
  if (status === 422) {
    // Not "photo 2 is too large": that is one photo of a post over its own cap (getPostPhotoProblem).
    return /(request body|file is) too large/i.test(JSON.stringify(data ?? ''));
  }
  // Every 500 the Go API produces is a Huma error object; anything else came from the proxy.
  return status === 500 && !(isRecord(data) && typeof data.status === 'number');
};

/**
 * Why the API refused the photos of a post. `index` is the 0-based position of the photo it names.
 * `totalTooLarge` is the whole request over the 10MB cap (see isUploadTooLarge).
 */
export type PostPhotoProblem =
  | { kind: 'tooMany' }
  | { kind: 'photoTooLarge'; index: number }
  | { kind: 'unsupportedImage'; index: number }
  | { kind: 'totalTooLarge' };

const photoIndex = (location: unknown): number | undefined => {
  const match = typeof location === 'string' ? /media\[(\d+)\]/.exec(location) : null;
  return match ? Number(match[1]) : undefined;
};

/**
 * Reads the reason out of a refused `POST /posts`. The API answers 422 with one entry in `errors` whose
 * `message` is a code (too_many_photos, photo_too_large, unsupported_image) and whose `location` is
 * `body.media[i]` for one photo. A part with a declared type other than JPEG, PNG or WebP is refused
 * earlier, by the framework, with "Invalid mime type" at `media[i]`.
 */
export const getPostPhotoProblem = (error: unknown): PostPhotoProblem | null => {
  if (isUploadTooLarge(error)) {
    return { kind: 'totalTooLarge' };
  }
  if (!isAxiosError(error) || error.response?.status !== 422) {
    return null;
  }
  const { data } = error.response;
  const details = isRecord(data) && Array.isArray(data.errors) ? data.errors.filter(isRecord) : [];
  for (const detail of details) {
    const index = photoIndex(detail.location);
    if (detail.message === 'too_many_photos') {
      return { kind: 'tooMany' };
    }
    if (detail.message === 'photo_too_large' && index !== undefined) {
      return { kind: 'photoTooLarge', index };
    }
    const mimeRefused = typeof detail.message === 'string' && detail.message.startsWith('Invalid mime type');
    if ((detail.message === 'unsupported_image' || mimeRefused) && index !== undefined) {
      return { kind: 'unsupportedImage', index };
    }
  }
  return null;
};

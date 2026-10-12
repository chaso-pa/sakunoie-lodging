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
    return /too large/i.test(JSON.stringify(data ?? ''));
  }
  // Every 500 the Go API produces is a Huma error object; anything else came from the proxy.
  return status === 500 && !(isRecord(data) && typeof data.status === 'number');
};

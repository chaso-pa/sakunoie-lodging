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

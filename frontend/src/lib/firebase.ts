import { type FirebaseApp, getApps, initializeApp } from 'firebase/app';
import { type Auth, browserSessionPersistence, connectAuthEmulator, initializeAuth } from 'firebase/auth';

type FirebaseEnv = {
  apiKey?: string;
  authDomain?: string;
  projectId?: string;
  authEmulatorHost?: string;
};

export type FirebaseSettings = {
  config: { apiKey: string; authDomain: string; projectId: string };
  /** `http://host:port` of the Auth Emulator, when used (dev/E2E). */
  emulatorUrl: string | null;
};

const DEMO_PROJECT_ID = 'demo-sakunoie';

/**
 * Firebase client settings from `NEXT_PUBLIC_FIREBASE_*`. With the Auth Emulator the API key may be
 * any string and the project defaults to `demo-sakunoie`; without it every value is required.
 */
export const firebaseSettingsFromEnv = (env: FirebaseEnv): FirebaseSettings => {
  const emulatorHost = env.authEmulatorHost?.trim() || '';
  if (emulatorHost) {
    const projectId = env.projectId?.trim() || DEMO_PROJECT_ID;
    return {
      config: {
        apiKey: env.apiKey?.trim() || 'demo-api-key',
        authDomain: env.authDomain?.trim() || `${projectId}.firebaseapp.com`,
        projectId
      },
      emulatorUrl: `http://${emulatorHost}`
    };
  }
  const apiKey = env.apiKey?.trim();
  const authDomain = env.authDomain?.trim();
  const projectId = env.projectId?.trim();
  if (!apiKey || !authDomain || !projectId) {
    throw new Error('Firebase is not configured: set NEXT_PUBLIC_FIREBASE_API_KEY, _AUTH_DOMAIN and _PROJECT_ID');
  }
  return { config: { apiKey, authDomain, projectId }, emulatorUrl: null };
};

let auth: Auth | undefined;

/**
 * The admin console's Firebase Auth instance (browser only).
 * - `initializeAuth` without a popup/redirect resolver: email/password sign-in needs no iframe,
 *   so the CSP needs no `frame-src`.
 * - Session persistence: the sign-in ends when the tab closes.
 * The `NEXT_PUBLIC_*` reads are literal so Next inlines them at build time; the same emulator host
 * drives the admin CSP `connect-src` in `src/proxy.ts`.
 */
export const getFirebaseAuth = (): Auth => {
  if (auth) return auth;
  const settings = firebaseSettingsFromEnv({
    apiKey: process.env.NEXT_PUBLIC_FIREBASE_API_KEY,
    authDomain: process.env.NEXT_PUBLIC_FIREBASE_AUTH_DOMAIN,
    projectId: process.env.NEXT_PUBLIC_FIREBASE_PROJECT_ID,
    authEmulatorHost: process.env.NEXT_PUBLIC_FIREBASE_AUTH_EMULATOR_HOST
  });
  const app: FirebaseApp = getApps()[0] ?? initializeApp(settings.config);
  auth = initializeAuth(app, { persistence: browserSessionPersistence });
  if (settings.emulatorUrl) connectAuthEmulator(auth, settings.emulatorUrl, { disableWarnings: true });
  return auth;
};

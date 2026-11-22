import { createFragmentTokenStore } from '@/features/registration/stores/fragment-token-store';

/** Edit-token session (plan Decision C): `#token=` → sessionStorage['sakunoie.manage.token']. */
const manage = createFragmentTokenStore('sakunoie.manage.token', 'token');

export const manageTokenStore = manage.store;
export const parseTokenFragment = manage.parse;
export const consumeTokenFragment = manage.consume;

import { beforeEach, describe, expect, it } from 'vitest';
import { useWizardStore } from '@/features/registration/stores/wizard-store';
import type { CreateRegistrationResult, FacilityPublic } from '@/features/registration/types/api';
import { emptyWizardValues } from '@/features/registration/types/schemas';

const facility = { code: 'saku-no-ie' } as FacilityPublic;
const result = { public_id: 'p', edit_token: 't' } as CreateRegistrationResult;
const store = () => useWizardStore.getState();

describe('wizard store idempotency key', () => {
  beforeEach(() => {
    store().open('other');
    store().startNew();
    store().open('saku-no-ie');
  });

  it('mints one key per session and reuses it for every retry', () => {
    const key = store().ensureIdempotencyKey();
    expect(key).toMatch(/^[A-Za-z0-9_.:-]{16,128}$/);
    expect(store().ensureIdempotencyKey()).toBe(key);
  });

  it('keeps the key when the submission is reported as already submitted', () => {
    const key = store().ensureIdempotencyKey();
    store().markAlreadySubmitted();
    expect(store().alreadySubmitted).toBe(true);
    expect(store().idempotencyKey).toBe(key);
    // Re-opening the same facility does not mint a new key either.
    store().open('saku-no-ie');
    expect(store().ensureIdempotencyKey()).toBe(key);
  });

  it('drops the draft at completion and the key only once the completion is acknowledged', () => {
    store().saveDraft(emptyWizardValues());
    const key = store().ensureIdempotencyKey();
    store().complete(result, facility);
    expect(store().draft).toBeNull();
    expect(store().idempotencyKey).toBe(key);
    store().acknowledgeCompletion();
    expect(store().idempotencyKey).toBeNull();
    expect(store().result?.registration.public_id).toBe('p');
  });

  it('discards the key when the guest explicitly starts a new registration', () => {
    const key = store().ensureIdempotencyKey();
    store().markAlreadySubmitted();
    store().startNew();
    expect(store().alreadySubmitted).toBe(false);
    expect(store().idempotencyKey).toBeNull();
    expect(store().ensureIdempotencyKey()).not.toBe(key);
  });
});

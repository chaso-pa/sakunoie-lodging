import { create } from 'zustand';
import type { CreateRegistrationResult, FacilityPublic } from '@/features/registration/types/api';
import type { WizardFormValues } from '@/features/registration/types/schemas';

/**
 * In-memory wizard session (no `persist` middleware: the draft holds passport numbers
 * and the result holds the one-time edit token, so neither may reach any storage).
 */
type WizardState = {
  facilityCode: string | null;
  draft: WizardFormValues | null;
  step: number;
  /** One key per wizard session; reused for every retry of the same submission. */
  idempotencyKey: string | null;
  /** Set on 409 `idempotency_conflict` or a replay without a token (`edit_link_unavailable`). */
  alreadySubmitted: boolean;
  result: { registration: CreateRegistrationResult; facility: FacilityPublic } | null;

  /** Starts a fresh session if the stored one belongs to another facility or has completed. */
  open: (facilityCode: string) => void;
  saveDraft: (values: WizardFormValues) => void;
  setStep: (step: number) => void;
  /** Returns the session key, minting it on the first submit only. */
  ensureIdempotencyKey: () => string;
  markAlreadySubmitted: () => void;
  /** Success: keep the result for the completion page and drop the draft (PII) at once. */
  complete: (registration: CreateRegistrationResult, facility: FacilityPublic) => void;
  /** Called once the completion page has shown the result: the key is no longer needed. */
  acknowledgeCompletion: () => void;
  /** Explicit "start a new registration": the only other way the key is discarded. */
  startNew: () => void;
};

const initial = {
  facilityCode: null,
  draft: null,
  step: 0,
  idempotencyKey: null,
  alreadySubmitted: false,
  result: null
} satisfies Partial<WizardState>;

export const useWizardStore = create<WizardState>()((set, get) => ({
  ...initial,
  open: (facilityCode) => {
    const state = get();
    if (state.facilityCode !== facilityCode || state.result !== null) {
      set({ ...initial, facilityCode });
    }
  },
  saveDraft: (values) => set({ draft: values }),
  setStep: (step) => set({ step }),
  ensureIdempotencyKey: () => {
    const existing = get().idempotencyKey;
    if (existing) return existing;
    const key = crypto.randomUUID();
    set({ idempotencyKey: key });
    return key;
  },
  markAlreadySubmitted: () => set({ alreadySubmitted: true }),
  complete: (registration, facility) =>
    set({ result: { registration, facility }, draft: null, step: 0, alreadySubmitted: false }),
  acknowledgeCompletion: () => set({ idempotencyKey: null }),
  startNew: () => set((state) => ({ ...initial, facilityCode: state.facilityCode }))
}));

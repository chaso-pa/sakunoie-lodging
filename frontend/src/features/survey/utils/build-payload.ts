import type { SurveyDefinition } from '@/config/survey-definition';

export type SurveyAnswerValue = number | string | string[] | undefined;
export type SurveyAnswers = Record<string, SurveyAnswerValue>;

export type SurveyPayload = {
  facility_code: string;
  survey_version: string;
  answers: Record<string, number | string | string[]>;
  /** Honeypot (always empty for people). */
  website: string;
};

/** Question codes failing client checks (UX only; the API validates again). */
export const surveyErrors = (def: SurveyDefinition, answers: SurveyAnswers): Record<string, 'required' | 'tooLong'> => {
  const errors: Record<string, 'required' | 'tooLong'> = {};
  for (const q of def.questions) {
    const v = answers[q.code];
    const empty = v === undefined || (typeof v === 'string' && v.trim() === '') || (Array.isArray(v) && v.length === 0);
    if (q.required && empty) errors[q.code] = 'required';
    if (q.type === 'text' && typeof v === 'string' && [...v.trim()].length > q.maxLength) errors[q.code] = 'tooLong';
  }
  return errors;
};

/** Drops unanswered questions and unknown codes; trims text. */
export const buildSurveyPayload = (
  def: SurveyDefinition,
  facilityCode: string,
  answers: SurveyAnswers,
  honeypot = ''
): SurveyPayload => {
  const out: SurveyPayload['answers'] = {};
  for (const q of def.questions) {
    const v = answers[q.code];
    if (v === undefined) continue;
    if (typeof v === 'string') {
      const s = v.trim();
      if (s) out[q.code] = s;
    } else if (Array.isArray(v)) {
      if (v.length) out[q.code] = v;
    } else {
      out[q.code] = v;
    }
  }
  return { facility_code: facilityCode, survey_version: def.version, answers: out, website: honeypot };
};

const MESSAGE_CODES = [
  'validation_error',
  'facility_unavailable',
  'rate_limited',
  'network_error',
  'internal_error'
] as const;
export type SurveyErrorKey = (typeof MESSAGE_CODES)[number] | 'unknown';

/** Maps an API error code to an `errors.*` message key. */
export const surveyErrorKey = (code: string | undefined, status = 0): SurveyErrorKey => {
  if (code && (MESSAGE_CODES as readonly string[]).includes(code)) return code as SurveyErrorKey;
  if (status >= 500) return 'internal_error';
  return 'unknown';
};

import { describe, expect, it } from 'vitest';
import { surveyDefinition } from '@/config/survey-definition';
import { buildSurveyPayload, surveyErrorKey, surveyErrors } from '@/features/survey/utils/build-payload';

describe('survey payload', () => {
  it('keeps answered questions only and trims text', () => {
    const p = buildSurveyPayload(surveyDefinition, 'saku-no-ie', {
      q01: 5,
      q02: undefined,
      q05: 'business',
      q08: [],
      q09: 'yes',
      q10: '  Nice  ',
      bogus: 'x'
    });
    expect(p).toEqual({
      facility_code: 'saku-no-ie',
      survey_version: 'v1',
      answers: { q01: 5, q05: 'business', q09: 'yes', q10: 'Nice' },
      website: ''
    });
  });

  it('flags missing required and over-long text', () => {
    expect(surveyErrors(surveyDefinition, {})).toEqual({ q01: 'required' });
    expect(surveyErrors(surveyDefinition, { q01: 3, q10: 'あ'.repeat(1001) })).toEqual({ q10: 'tooLong' });
    expect(surveyErrors(surveyDefinition, { q01: 3, q10: 'あ'.repeat(1000) })).toEqual({});
  });

  it('maps error codes to messages', () => {
    expect(surveyErrorKey('facility_unavailable', 422)).toBe('facility_unavailable');
    expect(surveyErrorKey('weird', 503)).toBe('internal_error');
    expect(surveyErrorKey(undefined, 400)).toBe('unknown');
  });
});

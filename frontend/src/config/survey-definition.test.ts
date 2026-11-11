import { readFileSync } from 'node:fs';
import path from 'node:path';
import { describe, expect, it } from 'vitest';
import { surveyDefinition } from '@/config/survey-definition';
import { routing } from '@/i18n/routing';

type BackendQuestion = { code: string; type: string; required?: boolean; options?: string[]; maxLength?: number };
const backend = JSON.parse(
  readFileSync(path.resolve(import.meta.dirname, '../../../backend/internal/services/surveydef/v1.json'), 'utf8')
) as { version: string; questions: BackendQuestion[] };

describe('survey definition', () => {
  it('matches the backend definition that validates submissions', () => {
    expect(surveyDefinition.version).toBe(backend.version);
    const shape = surveyDefinition.questions.map((q) => ({
      code: q.code,
      type: q.type,
      required: q.required ?? false,
      options: 'options' in q ? q.options.map((o) => o.value) : undefined,
      maxLength: 'maxLength' in q ? q.maxLength : undefined
    }));
    expect(shape).toEqual(
      backend.questions.map((q) => ({
        code: q.code,
        type: q.type,
        required: q.required ?? false,
        options: q.options,
        maxLength: q.maxLength
      }))
    );
  });

  it('has 10 questions q01..q10 labelled in every locale', () => {
    expect(surveyDefinition.questions.map((q) => q.code)).toEqual(
      Array.from({ length: 10 }, (_, i) => `q${String(i + 1).padStart(2, '0')}`)
    );
    for (const q of surveyDefinition.questions) {
      for (const locale of routing.locales) {
        expect(q.label[locale]?.trim(), `${q.code} ${locale}`).toBeTruthy();
        if ('options' in q)
          for (const o of q.options) expect(o.label[locale]?.trim(), `${q.code}.${o.value}`).toBeTruthy();
      }
    }
  });
});

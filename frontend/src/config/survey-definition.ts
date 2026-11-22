import v1 from '@content/surveys/v1.json';
import type { LocalizedText } from '@/config/site';

/**
 * Survey questions and labels (`content/surveys/v1.json`). Codes, types, options and required flags
 * mirror the backend definition `backend/internal/services/surveydef/v1.json`, which validates
 * submissions; `src/config/survey-definition.test.ts` checks the two stay in sync.
 */
export type SurveyOption = { value: string; label: LocalizedText };

export type SurveyQuestion =
  | { code: string; type: 'rating_1_5'; required?: boolean; label: LocalizedText }
  | {
      code: string;
      type: 'single_choice' | 'multi_choice';
      required?: boolean;
      label: LocalizedText;
      options: SurveyOption[];
    }
  | { code: string; type: 'text'; required?: boolean; label: LocalizedText; maxLength: number };

export type SurveyDefinition = { version: string; questions: SurveyQuestion[] };

export const surveyDefinition = v1 as SurveyDefinition;

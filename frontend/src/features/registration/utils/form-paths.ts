import type { GetInputPropsReturnType } from '@mantine/form';

/** The slice of a Mantine form that path-based field components need. */
export type PathForm = {
  getInputProps(path: string, options?: { type?: 'input' | 'checkbox' }): GetInputPropsReturnType;
  setFieldValue(path: string, value: unknown): void;
};

/** Stable DOM id for a form path, used by labels and the validation summary links. */
export const fieldId = (path: string) => `field-${path.replace(/\./g, '-')}`;

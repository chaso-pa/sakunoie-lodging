import { Text } from '@mantine/core';
import { IconCheck } from '@tabler/icons-react';
import classes from '@/features/registration/components/FormStepper.module.scss';

type FormStepperProps = {
  labels: string[];
  active: number;
  ariaLabel: string;
  /** e.g. "Step 2 of 4"; announced alongside the current label. */
  stepOfLabel: string;
};

/** Numbered 1–4 progress indicator; labels collapse to the current one on narrow screens. */
export function FormStepper({ labels, active, ariaLabel, stepOfLabel }: FormStepperProps) {
  return (
    <nav aria-label={ariaLabel}>
      <ol className={classes.list}>
        {labels.map((label, index) => {
          const state = index < active ? 'done' : index === active ? 'current' : 'todo';
          return (
            <li
              key={label}
              className={classes.item}
              data-state={state}
              aria-current={state === 'current' ? 'step' : undefined}
            >
              <span className={classes.marker} aria-hidden>
                {state === 'done' ? <IconCheck size={16} /> : index + 1}
              </span>
              <span className={classes.label}>{label}</span>
            </li>
          );
        })}
      </ol>
      <Text className={classes.current} fw={600} mt='xs'>
        {stepOfLabel}: {labels[active]}
      </Text>
    </nav>
  );
}

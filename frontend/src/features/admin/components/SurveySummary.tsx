import { Badge, Card, Group, Progress, SimpleGrid, Stack, Text, Title } from '@mantine/core';
import { IconEyeOff } from '@tabler/icons-react';
import type { SurveyQuestion } from '@/config/survey-definition';
import type { SurveyQuestionSummary } from '@/features/admin/types/api';

const typeLabel: Record<SurveyQuestionSummary['type'], string> = {
  rating_1_5: '5段階評価',
  single_choice: '単一選択',
  multi_choice: '複数選択',
  text: '自由記述'
};

type Props = { summary: SurveyQuestionSummary; question?: SurveyQuestion; index: number; minResponses: number };

/** One question's aggregates. Counts carry numbers as text, bars are decoration only. */
export function SurveySummary({ summary, question, index, minResponses }: Props) {
  const label = question?.label.ja ?? summary.code;
  const optionLabel = (value: string) => {
    if (summary.type === 'rating_1_5') return `${value}`;
    if (question && 'options' in question) return question.options.find((o) => o.value === value)?.label.ja ?? value;
    return value;
  };
  const max = Math.max(1, ...(summary.counts ?? []).map((c) => c.count));

  return (
    <Card padding='lg' component='section' aria-labelledby={`sq-${summary.code}`}>
      <Stack gap='sm'>
        <Group justify='space-between' align='flex-start' wrap='nowrap'>
          <Title order={2} fz='md' id={`sq-${summary.code}`}>
            {index}. {label}
          </Title>
          <Badge variant='light' color='forest' style={{ flexShrink: 0 }}>
            {typeLabel[summary.type]}
          </Badge>
        </Group>
        <Text size='sm' c='dimmed'>
          回答数 {summary.answered}件
        </Text>
        {summary.type === 'text' ? (
          <Text size='sm'>記入あり {summary.answered}件。自由記述の内容は個人の特定を避けるため表示しません。</Text>
        ) : summary.suppressed ? (
          <Group gap={6} wrap='nowrap'>
            <IconEyeOff size={18} aria-hidden />
            <Text size='sm'>回答が{minResponses}件未満のため、集計値を表示しません。</Text>
          </Group>
        ) : (
          <>
            {summary.mean !== null ? (
              <Text fw={700} fz='xl'>
                平均 {summary.mean.toFixed(2)} / 5
              </Text>
            ) : null}
            <SimpleGrid cols={1} spacing={6}>
              {(summary.counts ?? []).map((c) => (
                <div key={c.value}>
                  <Group justify='space-between' gap='xs' wrap='nowrap'>
                    <Text size='sm'>
                      {summary.type === 'rating_1_5' ? `${optionLabel(c.value)} 点` : optionLabel(c.value)}
                    </Text>
                    <Text size='sm' fw={600}>
                      {c.count}件
                    </Text>
                  </Group>
                  <Progress value={(c.count / max) * 100} color='forest' size='sm' aria-hidden />
                </div>
              ))}
            </SimpleGrid>
          </>
        )}
      </Stack>
    </Card>
  );
}

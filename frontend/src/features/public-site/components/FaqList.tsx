import { Accordion, AccordionControl, AccordionItem, AccordionPanel, Stack, Title } from '@mantine/core';
import { MarkdownContent } from '@/features/public-site/components/MarkdownContent';
import type { PageFrontMatter } from '@/features/public-site/utils/load-content';

type Props = { categories: PageFrontMatter<'faq'>['categories'] };

/** Categorized FAQ (DESIGN §5.5). Answers are Markdown. */
export function FaqList({ categories }: Props) {
  return (
    <Stack gap='xl'>
      {categories.map((category, ci) => (
        <section key={category.title} aria-labelledby={`faq-cat-${ci}`}>
          <Title order={2} id={`faq-cat-${ci}`} fz={{ base: 20, md: 24 }} mb='sm'>
            {category.title}
          </Title>
          <Accordion variant='separated' radius='md' multiple>
            {category.items.map((item, i) => (
              <AccordionItem key={item.q} value={`${ci}-${i}`}>
                <AccordionControl>{item.q}</AccordionControl>
                <AccordionPanel>
                  <MarkdownContent>{item.a}</MarkdownContent>
                </AccordionPanel>
              </AccordionItem>
            ))}
          </Accordion>
        </section>
      ))}
    </Stack>
  );
}

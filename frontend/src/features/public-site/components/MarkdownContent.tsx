import {
  Anchor,
  Blockquote,
  List,
  ListItem,
  Table,
  TableScrollContainer,
  TableTbody,
  TableTd,
  TableTh,
  TableThead,
  TableTr,
  Text,
  Title
} from '@mantine/core';
import Markdown, { type Components } from 'react-markdown';
import remarkGfm from 'remark-gfm';

/**
 * Renders version-controlled Markdown on the server. Raw HTML is not rendered (react-markdown's
 * default), so content cannot inject scripts. Headings start at h2: the page owns the h1. GFM adds
 * tables (e.g. the terms' 別表). Server component: use Mantine's flat exports (`TableTr`, not `Table.Tr`).
 */
const components: Components = {
  h1: ({ children }) => (
    <Title order={2} fz={{ base: 22, md: 26 }} mt='xl' mb='sm'>
      {children}
    </Title>
  ),
  h2: ({ children }) => (
    <Title order={2} fz={{ base: 22, md: 26 }} mt='xl' mb='sm'>
      {children}
    </Title>
  ),
  h3: ({ children }) => (
    <Title order={3} fz={{ base: 18, md: 20 }} mt='lg' mb='xs'>
      {children}
    </Title>
  ),
  p: ({ children }) => <Text mb='md'>{children}</Text>,
  ul: ({ children }) => (
    <List mb='md' spacing='xs'>
      {children}
    </List>
  ),
  ol: ({ children }) => (
    <List type='ordered' mb='md' spacing='xs'>
      {children}
    </List>
  ),
  li: ({ children }) => <ListItem>{children}</ListItem>,
  blockquote: ({ children }) => (
    <Blockquote color='terracotta' mb='md' p='md'>
      {children}
    </Blockquote>
  ),
  table: ({ children }) => (
    <TableScrollContainer minWidth={320}>
      <Table withTableBorder withColumnBorders mb='md' bg='var(--sakunoie-surface)'>
        {children}
      </Table>
    </TableScrollContainer>
  ),
  thead: ({ children }) => <TableThead>{children}</TableThead>,
  tbody: ({ children }) => <TableTbody>{children}</TableTbody>,
  tr: ({ children }) => <TableTr>{children}</TableTr>,
  th: ({ children }) => <TableTh scope='col'>{children}</TableTh>,
  td: ({ children }) => <TableTd>{children}</TableTd>,
  a: ({ href, children }) => {
    const external = !!href && /^https?:\/\//.test(href);
    return (
      <Anchor href={href} {...(external ? { target: '_blank', rel: 'noopener noreferrer' } : {})}>
        {children}
      </Anchor>
    );
  },
  img: () => null
};

export function MarkdownContent({ children }: { children: string }) {
  return (
    <div>
      <Markdown components={components} remarkPlugins={[remarkGfm]}>
        {children}
      </Markdown>
    </div>
  );
}

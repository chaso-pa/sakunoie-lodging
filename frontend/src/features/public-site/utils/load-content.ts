import { readFile } from 'node:fs/promises';
import path from 'node:path';
import matter from 'gray-matter';
import { cache } from 'react';
import { z } from 'zod';
import { type Locale, routing } from '@/i18n/routing';

/**
 * Server-only reader for `content/{locale}/{page}.md` (front matter + Markdown body). The files are
 * shipped with the standalone build through `outputFileTracingIncludes` in `next.config.ts`.
 */

export const pageSlugs = ['home', 'facilities', 'guide', 'faq', 'rates', 'terms', 'privacy'] as const;
export type PageSlug = (typeof pageSlugs)[number];

// Quoted in the files; js-yaml would otherwise turn an unquoted date into a Date.
const isoDate = z.string().regex(/^\d{4}-\d{2}-\d{2}$/);

const baseSchema = z.object({
  title: z.string().min(1),
  description: z.string().min(1),
  version: isoDate,
  effectiveDate: isoDate,
  placeholder: z.boolean(),
  todo: z.string().optional()
});

const featureIcon = z.enum(['door', 'users', 'map']);

export const frontMatterSchemas = {
  home: baseSchema.extend({
    lead: z.string().min(1),
    stayType: z.object({ title: z.string().min(1), body: z.string().min(1) }),
    features: z.array(z.object({ icon: featureIcon, title: z.string().min(1), body: z.string().min(1) })).length(3)
  }),
  facilities: baseSchema.extend({
    sections: z
      .array(
        z.object({
          id: z.enum(['room', 'shared', 'amenities', 'rules', 'access']),
          title: z.string().min(1),
          body: z.string().min(1)
        })
      )
      .min(1)
  }),
  guide: baseSchema,
  faq: baseSchema.extend({
    categories: z
      .array(
        z.object({
          title: z.string().min(1),
          items: z.array(z.object({ q: z.string().min(1), a: z.string().min(1) })).min(1)
        })
      )
      .min(1)
  }),
  rates: baseSchema.extend({
    disclaimer: z.string().min(1),
    rates: z.array(z.object({ label: z.string().min(1), price: z.string().min(1), note: z.string().optional() })).min(1)
  }),
  terms: baseSchema,
  privacy: baseSchema
} satisfies Record<PageSlug, z.ZodType>;

export type PageFrontMatter<S extends PageSlug> = z.infer<(typeof frontMatterSchemas)[S]>;
export type PageContent<S extends PageSlug> = { meta: PageFrontMatter<S>; body: string; locale: Locale };

export const contentDir = () => path.join(process.cwd(), 'content');

export const parsePage = <S extends PageSlug>(slug: S, raw: string): { meta: PageFrontMatter<S>; body: string } => {
  const { data, content } = matter(raw);
  const result = frontMatterSchemas[slug].safeParse(data);
  if (!result.success) {
    throw new Error(`content: invalid front matter in ${slug}.md: ${z.prettifyError(result.error)}`);
  }
  return { meta: result.data as PageFrontMatter<S>, body: content.trim() };
};

const readPage = async <S extends PageSlug>(locale: Locale, slug: S): Promise<PageContent<S>> => {
  const file = path.join(contentDir(), locale, `${slug}.md`);
  try {
    return { ...parsePage(slug, await readFile(file, 'utf8')), locale };
  } catch (error) {
    if ((error as NodeJS.ErrnoException).code !== 'ENOENT' || locale === routing.defaultLocale) throw error;
    // Missing translation: fall back to Japanese and say so in the logs (the content test fails too).
    console.warn(`content: ${locale}/${slug}.md is missing; falling back to ${routing.defaultLocale}`);
    return readPage(routing.defaultLocale, slug);
  }
};

/** Loads one page, memoized per request. */
export const loadPage = cache(readPage) as <S extends PageSlug>(locale: Locale, slug: S) => Promise<PageContent<S>>;

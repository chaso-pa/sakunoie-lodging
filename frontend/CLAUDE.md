# next-template — Claude Code Project Configuration

This project follows the [Bulletproof React](https://github.com/alan2207/bulletproof-react) architecture adapted for Next.js App Router.

## Stack

- **Framework**: Next.js (App Router)
- **UI**: Mantine v9
- **Styling**: SCSS Modules + PostCSS
- **Database**: API via backend (no direct DB access)
- **Linter/Formatter**: Biome
- **Language**: TypeScript (strict)

## File Naming Conventions

| File type | Convention | Example |
|---|---|---|
| React components | `PascalCase.tsx` | `UserProfile.tsx` |
| Hooks | `use-kebab-case.ts` | `use-discussions.ts` |
| Utilities / Config / Lib / Stores / Types | `kebab-case.ts` | `api-client.ts`, `paths.ts` |
| SCSS Modules | same as component | `UserProfile.module.scss` |

All **folders** under `src/` must be `kebab-case`.

## Directory Structure

```
src/
  app/            # Next.js App Router: layouts, pages, route segments
  assets/         # Static files (images, fonts, icons)
  components/     # Shared UI components reusable across the entire app
  config/         # Global config and environment variables
    paths.ts      # Centralized route path definitions (no hardcoded strings elsewhere)
  features/       # Feature-based self-contained modules (primary code home)
  hooks/          # Shared React hooks
  lib/            # Pre-configured library instances (e.g., axios api-client)
  stores/         # Global Zustand stores
  testing/        # Test utilities, mocks, and helpers
  types/          # Shared TypeScript type definitions
  utils/          # Shared utility functions
```

### Feature Module Structure

Each feature under `src/features/<name>/` is self-contained:

```
src/features/<name>/
  api/            # API calls + TanStack Query hooks
  assets/         # Feature-scoped static files
  components/     # Components used only within this feature (PascalCase.tsx)
  hooks/          # Feature-scoped custom hooks
  stores/         # Zustand stores scoped to this feature
  types/          # TypeScript types for this feature
  utils/          # Utility functions for this feature
  index.ts        # Public barrel export — only expose what other layers need
```

## Import Conventions

Always use the `@/` alias. Never use relative paths like `../../components/...`.

```ts
// Good
import { Button } from '@/components/Button';
import { paths } from '@/config/paths';
import { useDiscussions } from '@/features/discussions';

// Bad
import { Button } from '../../../components/Button';
```

## Unidirectional Dependency Flow

```
shared (components, hooks, utils, lib, stores, types, config)
    ↓
features (src/features/*)
    ↓
app (src/app/)
```

Rules:
- `src/features/*` can import from shared layer
- `src/app/*` can import from features and shared layer
- **`src/features/foo` cannot import from `src/features/bar`** (no cross-feature imports)
- Shared layer cannot import from features or app

## Route Paths

All route strings are defined in `src/config/paths.ts`. Never hardcode path strings in components or API hooks.

```ts
// Good
import { paths } from '@/config/paths';
<Link href={paths.home.getHref()} />

// Bad
<Link href="/" />
```

## API Layer (when adding server communication)

Each feature's `api/` directory follows a three-part pattern per file:

1. **Zod schema + TypeScript type**
2. **Fetcher function** using the shared `@/lib/api-client` axios instance
3. **TanStack Query hook** (`useQuery` or `useMutation`)

```ts
// features/discussions/api/create-discussion.ts
import { z } from 'zod';
import { api } from '@/lib/api-client';
import { useMutation, useQueryClient } from '@tanstack/react-query';

export const createDiscussionInputSchema = z.object({ title: z.string().min(1) });
export type CreateDiscussionInput = z.infer<typeof createDiscussionInputSchema>;

const createDiscussion = (data: CreateDiscussionInput) =>
  api.post<Discussion>('/discussions', data);

export const useCreateDiscussion = () => {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: createDiscussion,
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['discussions'] }),
  });
};
```

## State Management

| State type | Tool |
|---|---|
| Local UI state | `useState` / `useReducer` |
| Global UI state | Zustand (`src/stores/`) |
| Server cache | TanStack Query |
| Form state | `@mantine/form` or `react-hook-form` + Zod |
| URL state | Next.js router / `useSearchParams` |

## Recommended Packages (when needed)

When adding server communication or global state:

```bash
npm install @tanstack/react-query zustand zod axios
npm install @tanstack/react-query-devtools --save-dev
```

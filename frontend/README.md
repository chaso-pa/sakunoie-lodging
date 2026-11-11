# next-template

Next.js starter template following [Bulletproof React](https://github.com/alan2207/bulletproof-react) architecture conventions.

## Stack

- **Next.js** (App Router) — framework
- **Mantine v8** — UI component library
- **SCSS Modules** + PostCSS — styling
- **Prisma** — database ORM
- **Biome** — linting & formatting
- **TypeScript** (strict mode)

## Getting Started

```bash
npm install
npm run dev
```

Open [http://localhost:3000](http://localhost:3000).

## Project Structure

```
src/
  app/            # Next.js App Router pages and layouts
  assets/         # Static files (images, fonts, icons)
  components/     # Shared UI components (used across features)
  config/
    paths.ts      # Centralized route path strings
  features/       # Feature-based modules (primary application code)
  hooks/          # Shared React hooks
  lib/            # Pre-configured library instances (e.g., api-client)
  stores/         # Global Zustand state stores
  testing/        # Test utilities and helpers
  types/          # Shared TypeScript types
  utils/          # Shared utility functions
```

### Feature Module

Each feature under `src/features/<name>/` is self-contained:

```
src/features/<name>/
  api/            # API calls + TanStack Query hooks
  components/     # Components scoped to this feature
  hooks/          # Hooks scoped to this feature
  stores/         # Zustand stores scoped to this feature
  types/          # Types scoped to this feature
  utils/          # Utils scoped to this feature
  index.ts        # Public barrel export
```

## Architecture Principles

### Unidirectional Dependency Flow

```
shared (components, hooks, utils, lib, stores, types, config)
    ↓
features
    ↓
app
```

- Features can import from the shared layer
- The app layer can import from features and shared
- **No cross-feature imports** — features are isolated from each other

### File Naming

- React components: `PascalCase.tsx`
- All other TypeScript files: `kebab-case.ts`
- Folders: `kebab-case`

### Imports

Always use `@/` alias. Relative imports are not allowed.

```ts
import { UserCard } from '@/components/UserCard';
import { paths } from '@/config/paths';
import { useAuth } from '@/features/auth';
```

### Route Paths

All paths are defined in `src/config/paths.ts`:

```ts
import { paths } from '@/config/paths';
href={paths.home.getHref()}
```

## Scripts

```bash
npm run dev        # Start dev server (Turbopack)
npm run build      # Production build
npm run lint       # Biome lint check
npm run format     # Biome format (write)
npm run db:generate  # Prisma client generation
```

## Reference

- [Bulletproof React](https://github.com/alan2207/bulletproof-react)
- [Mantine](https://mantine.dev)
- [Next.js App Router](https://nextjs.org/docs/app)
- [Prisma](https://www.prisma.io/docs)
- [Biome](https://biomejs.dev)

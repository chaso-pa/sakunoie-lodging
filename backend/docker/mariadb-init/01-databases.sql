-- Databases inside the throwaway compose MariaDB only.
--   lodging_migrate  target of `prisma migrate dev` (reset freely)
--   lodging_shadow   Prisma shadow database
--   lodging_test     Go integration tests
--   lodging_e2e      Playwright E2E
CREATE DATABASE IF NOT EXISTS lodging_migrate CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
CREATE DATABASE IF NOT EXISTS lodging_shadow CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
CREATE DATABASE IF NOT EXISTS lodging_test CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
CREATE DATABASE IF NOT EXISTS lodging_e2e CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

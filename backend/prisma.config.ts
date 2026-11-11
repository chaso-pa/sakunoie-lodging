import "dotenv/config";
import { defineConfig, env } from "prisma/config";

// Safety guard (plan pre-mortem P1).
//
// `.env` points DATABASE_URL at the shared dev server, where only
// `migrate deploy`, `migrate status` and `db execute` (seed) are allowed.
// Commands that reset databases or create shadow databases must run against
// the throwaway compose MariaDB (backend/compose.test.yaml), with both URLs
// set inline. dotenv never overrides variables that are already set.
const COMPOSE_HOST = "127.0.0.1";
const COMPOSE_PORT = process.env.TEST_DB_PORT || "33306";
const COMPOSE_DBS = new Set(["lodging_migrate", "lodging_shadow", "lodging_test", "lodging_e2e"]);

function isComposeUrl(raw: string | undefined): boolean {
	if (!raw) return false;
	try {
		const u = new URL(raw);
		const db = decodeURIComponent(u.pathname.replace(/^\//, ""));
		return u.hostname === COMPOSE_HOST && u.port === COMPOSE_PORT && COMPOSE_DBS.has(db);
	} catch {
		return false;
	}
}

const args = process.argv.slice(2);
const devOnlyCommand =
	(args.includes("migrate") && args.some((a) => a === "dev" || a === "diff" || a === "reset")) ||
	(args.includes("db") && args.includes("push"));

if (
	devOnlyCommand &&
	!(isComposeUrl(process.env.DATABASE_URL) && isComposeUrl(process.env.SHADOW_DATABASE_URL))
) {
	throw new Error(
		"refusing: dev-only Prisma command against non-compose DB " +
			`(DATABASE_URL and SHADOW_DATABASE_URL must both be mysql://…@${COMPOSE_HOST}:${COMPOSE_PORT}/lodging_*)`,
	);
}

export default defineConfig({
	schema: "prisma/schema.prisma",
	migrations: {
		path: "prisma/migrations",
	},
	datasource: {
		url: env("DATABASE_URL"),
		// Optional: only set (inline, compose only) for migrate dev/diff.
		// env() throws when a variable is unset, so read process.env directly.
		...(process.env.SHADOW_DATABASE_URL
			? { shadowDatabaseUrl: process.env.SHADOW_DATABASE_URL }
			: {}),
	},
});

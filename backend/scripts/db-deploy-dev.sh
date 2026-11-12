#!/usr/bin/env bash
# Apply migrations and the seed to the shared dev database. Additive only.
#   bash scripts/db-deploy-dev.sh
#
# Reads DATABASE_URL from backend/.env (an already-exported DATABASE_URL takes
# precedence, as with dotenv) and refuses anything other than exactly
# lodging-master-dev on the shared server. Never runs migrate dev/diff/reset
# or db push, so Prisma never creates a shadow database there.
set -euo pipefail
cd "$(dirname "$0")/.."
source scripts/assert-safe-db.sh

if [[ -z "${DATABASE_URL:-}" && -f .env ]]; then
	DATABASE_URL="$(sed -nE 's/^[[:space:]]*DATABASE_URL[[:space:]]*=[[:space:]]*"?([^"]*)"?[[:space:]]*$/\1/p' .env | tail -n 1)"
fi
[[ -n "${DATABASE_URL:-}" ]] || {
	echo "db-deploy-dev: DATABASE_URL is not set (backend/.env)" >&2
	exit 1
}

assert_safe_db bootstrap "$DATABASE_URL"
[[ "$DB_HOST:$DB_PORT/$DB_NAME" == "$SAFE_DB_SHARED_HOST:$SAFE_DB_SHARED_PORT/$SAFE_DB_SHARED_NAME" ]] ||
	_safe_db_fail "db-deploy-dev only targets $SAFE_DB_SHARED_NAME on the shared server (use db-reset-test.sh for compose)"

cnf="$(mktemp)"
trap 'rm -f "$cnf"' EXIT
write_mysql_cnf "$DATABASE_URL" "$cnf"

shadow_dbs() {
	mysql --defaults-extra-file="$cnf" -N -e "SHOW DATABASES LIKE 'prisma\\_migrate\\_shadow\\_db%'" | sort
}
before="$(shadow_dbs)"

# 1. The only DDL issued on the shared server outside Prisma migrations.
mysql --defaults-extra-file="$cnf" -e "CREATE DATABASE IF NOT EXISTS \`${SAFE_DB_SHARED_NAME}\` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;"

# 2-3. No shadow database is used by deploy or db execute.
export DATABASE_URL
env -u SHADOW_DATABASE_URL npx --no-install prisma migrate deploy
env -u SHADOW_DATABASE_URL npx --no-install prisma db execute --file prisma/seed.sql

# 4. Read-only check: no new Prisma shadow database appeared.
after="$(shadow_dbs)"
if [[ "$before" != "$after" ]]; then
	echo "db-deploy-dev: WARNING: prisma_migrate_shadow_db_* set changed on the shared server:" >&2
	diff <(echo "$before") <(echo "$after") >&2 || true
	exit 1
fi
echo "db-deploy-dev: ${SAFE_DB_SHARED_NAME} migrated and seeded; no new shadow databases"

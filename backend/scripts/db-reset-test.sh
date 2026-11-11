#!/usr/bin/env bash
# Recreate a throwaway compose database and apply migrations + seed.
#   bash scripts/db-reset-test.sh <lodging_test|lodging_e2e>
# Requires: docker compose -f compose.test.yaml up -d --wait
set -euo pipefail
cd "$(dirname "$0")/.."
source scripts/assert-safe-db.sh

db="${1:-}"
case "$db" in
lodging_test | lodging_e2e) ;;
*)
	echo "usage: $0 <lodging_test|lodging_e2e>" >&2
	exit 2
	;;
esac

# Fixed throwaway credentials of compose.test.yaml (safe to commit).
url="mysql://root:test@127.0.0.1:${TEST_DB_PORT:-33306}/${db}"
assert_safe_db destructive "$url"

cnf="$(mktemp)"
trap 'rm -f "$cnf"' EXIT
write_mysql_cnf "$url" "$cnf"

mysql --defaults-extra-file="$cnf" -e "DROP DATABASE IF EXISTS \`${db}\`; CREATE DATABASE \`${db}\` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;"

# Inline variables win over backend/.env (dotenv never overrides).
DATABASE_URL="$url" npx --no-install prisma migrate deploy
DATABASE_URL="$url" npx --no-install prisma db execute --file prisma/seed.sql
echo "db-reset-test: ${db} recreated, migrated and seeded"

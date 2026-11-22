#!/usr/bin/env bash
# Database safety guard (plan pre-mortem P1). Source this file first in every
# script that touches a database:
#
#   source "$(dirname "$0")/assert-safe-db.sh"
#   assert_safe_db <op> "$URL"        # exits 1 with the reason if not allowed
#   write_mysql_cnf "$URL" "$CNF"     # 0600 [client] file for --defaults-extra-file
#
# <op> is one of:
#   destructive  drop/create/truncate/reset/migrate dev. Compose DB only.
#   deploy|seed|bootstrap
#                additive. Compose DB, or exactly the shared dev DB.
#
# Allowed targets:
#   compose     127.0.0.1:${TEST_DB_PORT:-33306} / lodging_(test|e2e|migrate|shadow)
#   shared dev  192.168.3.100:23306 / lodging-master-dev  (deploy|seed|bootstrap only)
#
# <url> may be a Prisma URL (mysql://user:pass@host:port/db?...) or a
# go-sql-driver DSN (user:pass@tcp(host:port)/db?...).
#
# Parsing uses bash builtins only, so the password never reaches the argv of
# a child process. Prefer sourcing over executing this file with a URL
# argument, because a command-line argument is visible in the process list.

SAFE_DB_COMPOSE_HOST="127.0.0.1"
SAFE_DB_COMPOSE_PORT="${TEST_DB_PORT:-33306}"
SAFE_DB_COMPOSE_NAME_RE='^lodging_(test|e2e|migrate|shadow)$'
SAFE_DB_SHARED_HOST="192.168.3.100"
SAFE_DB_SHARED_PORT="23306"
SAFE_DB_SHARED_NAME="lodging-master-dev"

_safe_db_fail() {
	echo "assert-safe-db: refusing: $*" >&2
	exit 1
}

_safe_db_urldecode() {
	local s="${1//+/ }"
	printf '%b' "${s//%/\\x}"
}

# Sets DB_USER, DB_PASS, DB_HOST, DB_PORT, DB_NAME from a URL or DSN.
parse_db_url() {
	local url="$1" rest userinfo hostport
	DB_USER="" DB_PASS="" DB_HOST="" DB_PORT="" DB_NAME=""
	if [[ "$url" =~ ^mysql://([^@/]*)@([^/]+)/([^?]*) ]]; then
		userinfo="${BASH_REMATCH[1]}"
		hostport="${BASH_REMATCH[2]}"
		DB_NAME="$(_safe_db_urldecode "${BASH_REMATCH[3]}")"
		DB_USER="$(_safe_db_urldecode "${userinfo%%:*}")"
		[[ "$userinfo" == *:* ]] && DB_PASS="$(_safe_db_urldecode "${userinfo#*:}")"
	elif [[ "$url" =~ ^(.*)@tcp\(([^\)]+)\)/([^?]*) ]]; then
		userinfo="${BASH_REMATCH[1]}"
		hostport="${BASH_REMATCH[2]}"
		DB_NAME="${BASH_REMATCH[3]}"
		DB_USER="${userinfo%%:*}"
		[[ "$userinfo" == *:* ]] && DB_PASS="${userinfo#*:}"
	else
		return 1
	fi
	rest="$hostport"
	DB_HOST="${rest%%:*}"
	if [[ "$rest" == *:* ]]; then DB_PORT="${rest##*:}"; else DB_PORT="3306"; fi
	[[ -n "$DB_HOST" && -n "$DB_NAME" && -n "$DB_USER" ]]
}

# assert_safe_db <op> <url>
assert_safe_db() {
	local op="$1" url="$2"
	case "$op" in
	destructive | deploy | seed | bootstrap) ;;
	*) _safe_db_fail "unknown op '$op' (expected destructive|deploy|seed|bootstrap)" ;;
	esac
	parse_db_url "$url" || _safe_db_fail "cannot parse database URL"

	if [[ "$DB_HOST" == "$SAFE_DB_COMPOSE_HOST" && "$DB_PORT" == "$SAFE_DB_COMPOSE_PORT" ]]; then
		[[ "$DB_NAME" =~ $SAFE_DB_COMPOSE_NAME_RE ]] ||
			_safe_db_fail "compose DB '$DB_NAME' is not one of lodging_(test|e2e|migrate|shadow)"
		return 0
	fi

	if [[ "$DB_HOST" == "$SAFE_DB_SHARED_HOST" && "$DB_PORT" == "$SAFE_DB_SHARED_PORT" ]]; then
		[[ "$DB_NAME" == "$SAFE_DB_SHARED_NAME" ]] ||
			_safe_db_fail "only '$SAFE_DB_SHARED_NAME' may be used on the shared server (got '$DB_NAME')"
		[[ "$op" != "destructive" ]] ||
			_safe_db_fail "destructive operations are never allowed on the shared server"
		return 0
	fi

	_safe_db_fail "host $DB_HOST:$DB_PORT is neither the compose DB ($SAFE_DB_COMPOSE_HOST:$SAFE_DB_COMPOSE_PORT) nor the shared dev DB"
}

# write_mysql_cnf <url> <file>: writes a mode-0600 [client] option file with
# user, password, host and port. The caller must delete it (use a trap).
write_mysql_cnf() {
	local url="$1" file="$2" pass
	parse_db_url "$url" || _safe_db_fail "cannot parse database URL"
	pass="${DB_PASS//\\/\\\\}"
	pass="${pass//\"/\\\"}"
	(
		umask 077
		printf '[client]\nuser="%s"\npassword="%s"\nhost="%s"\nport=%s\n' \
			"$DB_USER" "$pass" "$DB_HOST" "$DB_PORT" >"$file"
	)
	chmod 600 "$file"
}

if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
	set -euo pipefail
	[[ $# -eq 2 ]] || {
		echo "usage: $0 <destructive|deploy|seed|bootstrap> <url>" >&2
		exit 2
	}
	assert_safe_db "$1" "$2"
	echo "assert-safe-db: ok ($1 on $DB_HOST:$DB_PORT/$DB_NAME)"
fi

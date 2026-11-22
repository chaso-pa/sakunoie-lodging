#!/usr/bin/env bash
# Local E2E stack for Playwright (plan Phase 6).
#
#   bash scripts/e2e-stack.sh start    # compose + reset lodging_e2e, Auth Emulator, API, Next build + standalone
#
# Cross-origin like production: Next on :3100 is built with NEXT_PUBLIC_API_BASE_URL=http://127.0.0.1:8180,
# so the browser calls the API directly and the API answers CORS for http://127.0.0.1:3100 only.
#   bash scripts/e2e-stack.sh stop     # stop only the processes this script started; compose down -v
#   bash scripts/e2e-stack.sh status
#
# Ports: web 3100, API 8180, Auth Emulator 9099, compose MariaDB 127.0.0.1:${TEST_DB_PORT:-33306}.
# It refuses to start when one of them is already bound, never touches :3000/:8080, and never
# kills a process it did not start (PIDs are kept in $E2E_STATE_DIR/pids).
#
# Optional env:
#   GORM_DATABASE_URL_E2E  API DSN (default: compose lodging_e2e; must be compose lodging_e2e)
#   E2E_STATE_DIR          logs, PIDs and the API binary (default: frontend/.e2e-stack)
#   E2E_PID_LOG            extra file that every started PID is appended to
#   E2E_SKIP_BUILD=1       reuse the existing .next build (it must have been built with the same env,
#                          including NEXT_PUBLIC_API_BASE_URL=http://127.0.0.1:8180)
#
# No password is passed on a command line: secrets are generated into exported variables.
set -euo pipefail

FRONTEND="$(cd "$(dirname "$0")/.." && pwd)"
ROOT="$(cd "$FRONTEND/.." && pwd)"
BACKEND="$ROOT/backend"
STATE="${E2E_STATE_DIR:-$FRONTEND/.e2e-stack}"
PIDS="$STATE/pids"

WEB_PORT=3100
API_PORT=8180
EMU_PORT=9099
DB_PORT="${TEST_DB_PORT:-33306}"
EMU_HOST="127.0.0.1:$EMU_PORT"
WEB_ORIGIN="http://127.0.0.1:$WEB_PORT"
API_ORIGIN="http://127.0.0.1:$API_PORT"
PROJECT_ID=demo-sakunoie
E2E_DSN_DEFAULT="root:test@tcp(127.0.0.1:${DB_PORT})/lodging_e2e?charset=utf8mb4&parseTime=True"

log() { printf 'e2e-stack: %s\n' "$*"; }
die() {
  printf 'e2e-stack: %s\n' "$*" >&2
  exit 1
}

port_bound() { ss -Htln "sport = :$1" | grep -q .; }

record_pid() {
  echo "$1 $2" >>"$PIDS"
  if [[ -n "${E2E_PID_LOG:-}" ]]; then
    echo "e2e-stack $2 pid=$1 $(date -Iseconds)" >>"$E2E_PID_LOG"
  fi
}

wait_http() {
  local url="$1" name="$2" tries="${3:-120}"
  for ((i = 0; i < tries; i++)); do
    if curl -fs -o /dev/null --max-time 2 "$url"; then return 0; fi
    sleep 1
  done
  die "$name did not become ready ($url); see $STATE/*.log"
}

# Starts "$@" in its own session (process group) so stop can signal the whole group.
spawn() {
  local name="$1" logfile="$2"
  shift 2
  setsid "$@" >"$logfile" 2>&1 </dev/null &
  local pid=$!
  record_pid "$pid" "$name"
  log "$name started (pid $pid)"
}

stop_stack() {
  if [[ -f "$PIDS" ]]; then
    # Reverse order: web, API, emulator.
    tac "$PIDS" | while read -r pid name; do
      [[ -n "$pid" ]] || continue
      if kill -0 "$pid" 2>/dev/null; then
        # Each process was started with setsid, so its PID is also its process-group ID.
        kill -TERM -- "-$pid" 2>/dev/null || kill -TERM "$pid" 2>/dev/null || true
        for _ in $(seq 1 20); do
          kill -0 "$pid" 2>/dev/null || break
          sleep 0.5
        done
        kill -KILL -- "-$pid" 2>/dev/null || true
        log "stopped $name (pid $pid)"
        if [[ -n "${E2E_PID_LOG:-}" ]]; then echo "e2e-stack stopped $name pid=$pid $(date -Iseconds)" >>"$E2E_PID_LOG"; fi
      fi
    done
    rm -f "$PIDS"
  fi
  if [[ -f "$STATE/compose-up" ]]; then
    docker compose -f "$BACKEND/compose.test.yaml" down -v
    rm -f "$STATE/compose-up"
    if [[ -n "${E2E_PID_LOG:-}" ]]; then echo "e2e-stack compose down -v $(date -Iseconds)" >>"$E2E_PID_LOG"; fi
  fi
  log "stopped"
}

start_stack() {
  mkdir -p "$STATE"
  chmod 700 "$STATE"
  [[ -s "$PIDS" ]] && die "a stack is already recorded in $PIDS; run '$0 stop' first"
  for p in "$WEB_PORT" "$API_PORT" "$EMU_PORT"; do
    port_bound "$p" && die "port $p is already in use by a process this script did not start"
  done
  : >"$PIDS"
  STARTED=0
  trap '[[ ${STARTED:-0} == 1 ]] || { log "start failed; stopping what was started"; stop_stack; }' EXIT

  # 1. Compose MariaDB + fresh lodging_e2e (destructive; compose only, guarded by assert-safe-db.sh).
  local e2e_dsn="${GORM_DATABASE_URL_E2E:-$E2E_DSN_DEFAULT}"
  [[ "$e2e_dsn" == *"@tcp(127.0.0.1:${DB_PORT})/lodging_e2e"* ]] ||
    die "GORM_DATABASE_URL_E2E must point at compose 127.0.0.1:${DB_PORT}/lodging_e2e"
  docker compose -f "$BACKEND/compose.test.yaml" up -d --wait
  touch "$STATE/compose-up"
  if [[ -n "${E2E_PID_LOG:-}" ]]; then echo "e2e-stack compose up $(date -Iseconds)" >>"$E2E_PID_LOG"; fi
  bash "$BACKEND/scripts/db-reset-test.sh" lodging_e2e

  # 2. Firebase Auth Emulator (Java). The demo- project needs no credentials.
  (cd "$FRONTEND" && spawn emulator "$STATE/emulator.log" firebase emulators:start --only auth --project "$PROJECT_ID")
  wait_http "http://$EMU_HOST/" "Auth Emulator" 90

  # 3. API on :8180 with per-run test keys (exported, never on argv) and raised limits.
  (cd "$BACKEND" && go build -o "$STATE/sakunoie-server" ./cmd/server)
  (
    cd "$BACKEND"
    # Never read backend/.env (shared dev values) in an E2E run.
    export SKIP_DOTENV=1 APP_ENV=test PORT="$API_PORT" GORM_DATABASE_URL="$e2e_dsn"
    PASSPORT_ENC_KEYS="1:$(openssl rand -base64 32)"
    IDEMPOTENCY_HMAC_KEY="$(openssl rand -base64 32)"
    export PASSPORT_ENC_KEYS IDEMPOTENCY_HMAC_KEY PASSPORT_ENC_ACTIVE_KEY_ID=1
    export FIREBASE_PROJECT_ID="$PROJECT_ID" FIREBASE_AUTH_EMULATOR_HOST="$EMU_HOST"
    export RATE_LIMIT_CREATE_PER_MIN=1000 RATE_LIMIT_MANAGE_PER_MIN=1000 RATE_LIMIT_SURVEY_PER_MIN=1000
    # The browser on :3100 calls :8180 directly. No proxy sets a client-IP header here (default X-Real-IP).
    export CORS_ALLOWED_ORIGINS="$WEB_ORIGIN" CLIENT_IP_HEADER=X-Real-IP BIND_ADDR=127.0.0.1
    export SURVEY_MIN_RESPONSES=1
    spawn api "$STATE/api.log" "$STATE/sakunoie-server"
  )
  wait_http "http://127.0.0.1:$API_PORT/ready" "API" 60

  # 4. Next production build (API origin and emulator host are inlined), then standalone on :3100.
  # With NEXT_PUBLIC_API_BASE_URL set, the /api rewrite is not built, so nothing goes through Next.
  export NEXT_PUBLIC_API_BASE_URL="$API_ORIGIN"
  unset API_PROXY_TARGET
  export NEXT_PUBLIC_FIREBASE_AUTH_EMULATOR_HOST="$EMU_HOST"
  export NEXT_PUBLIC_FIREBASE_PROJECT_ID="$PROJECT_ID"
  export NEXT_TELEMETRY_DISABLED=1
  if [[ "${E2E_SKIP_BUILD:-}" != 1 ]]; then
    # Next 16.4 may stop once with a SecurityFatalError (advisory check); retry once.
    (cd "$FRONTEND" && npx --no-install next build >"$STATE/next-build.log" 2>&1) ||
      (cd "$FRONTEND" && npx --no-install next build >"$STATE/next-build.log" 2>&1) ||
      die "next build failed; see $STATE/next-build.log"
  fi
  local standalone="$FRONTEND/.next/standalone"
  [[ -f "$standalone/server.js" ]] || die "missing $standalone/server.js (next build with output: 'standalone')"
  rm -rf "$standalone/public" "$standalone/.next/static"
  cp -a "$FRONTEND/public" "$standalone/public"
  cp -a "$FRONTEND/.next/static" "$standalone/.next/static"
  (
    cd "$standalone"
    export PORT="$WEB_PORT" HOSTNAME=127.0.0.1 NODE_ENV=production
    spawn web "$STATE/web.log" node server.js
  )
  wait_http "http://127.0.0.1:$WEB_PORT/ja" "Next standalone" 60

  STARTED=1
  log "ready: web http://127.0.0.1:$WEB_PORT, API :$API_PORT, emulator :$EMU_PORT (logs in $STATE)"
}

case "${1:-}" in
start) start_stack ;;
stop) stop_stack ;;
status)
  [[ -f "$PIDS" ]] || {
    log "not running"
    exit 0
  }
  while read -r pid name; do
    if kill -0 "$pid" 2>/dev/null; then log "$name pid $pid running"; else log "$name pid $pid gone"; fi
  done <"$PIDS"
  ;;
*)
  echo "usage: $0 start|stop|status" >&2
  exit 2
  ;;
esac

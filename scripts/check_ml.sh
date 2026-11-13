#!/usr/bin/env bash
# Full local verification; resources are disposable and never target production.
set -euo pipefail
cd "$(dirname "$0")/.."
check_dir="$(mktemp -d "${TMPDIR:-/tmp}/rhythm-ml-check.XXXXXX")"
db_name="rhythm-ml-check-db-$$"
redis_name="rhythm-ml-check-redis-$$"
backend_pid=""
cleanup() {
 status=$?
 if [ "$status" -ne 0 ] && [ -f "$check_dir/backend.log" ]; then tail -40 "$check_dir/backend.log"; fi
 if [ -n "$backend_pid" ]; then kill "$backend_pid" 2>/dev/null || true; wait "$backend_pid" 2>/dev/null || true; fi
 docker rm -f "$db_name" "$redis_name" >/dev/null 2>&1 || true
 rm -rf "$check_dir"
 exit "$status"
}
trap cleanup EXIT
export GOCACHE="${GOCACHE:-$check_dir/go-cache}"
python3 -m unittest discover -s ml -v
python3 ml/train.py --check
go test -count=1 ./...
go test -race ./internal/features/priorities/...
go vet ./...
npm --prefix frontend test
npm --prefix frontend run build
go build -buildvcs=false -o "$check_dir/backend" ./cmd/server
docker run -d --name "$db_name" -e POSTGRES_PASSWORD=ml-check-only -e POSTGRES_DB=rhythm_check -p 127.0.0.1::5432 postgres:16-alpine >/dev/null
docker run -d --name "$redis_name" -p 127.0.0.1::6379 redis:7-alpine >/dev/null
for ((i=0;i<150;i++)); do
 if docker exec "$db_name" pg_isready -h 127.0.0.1 -U postgres -d rhythm_check >/dev/null 2>&1; then break; fi
 sleep .2
done
python3 - "$check_dir/schema.sql" <<'PY'
from pathlib import Path
import sys
Path(sys.argv[1]).write_text('\n'.join(p.read_text() for p in sorted(Path('migrations').glob('*.up.sql'))))
PY
docker exec -i "$db_name" psql -U postgres -d rhythm_check -v ON_ERROR_STOP=1 < "$check_dir/schema.sql" >/dev/null
db_port="$(docker port "$db_name" 5432/tcp)"; db_port="${db_port##*:}"
redis_port="$(docker port "$redis_name" 6379/tcp)"; redis_port="${redis_port##*:}"
http_port="$(python3 - <<'PY'
import socket
with socket.socket() as s:
 s.bind(('127.0.0.1',0));print(s.getsockname()[1])
PY
)"
start_backend() {
 env TIME_ZONE=UTC JWT_ACCESS_SECRET=ml-check-access-secret-at-least-32-characters JWT_REFRESH_SECRET=ml-check-refresh-secret-at-least-32-characters JWT_ACCESS_TTL=15m JWT_REFRESH_TTL=720h \
 POSTGRES_HOST=127.0.0.1 POSTGRES_PORT="$db_port" POSTGRES_USER=postgres POSTGRES_PASSWORD=ml-check-only POSTGRES_NAME=rhythm_check POSTGRES_TIMEOUT=5s \
 REDIS_ADDR="127.0.0.1:$redis_port" HTTP_ADDR="127.0.0.1:$http_port" LOGGER_FOLDER="$check_dir/logs" LOGGER_LEVEL=INFO \
 "$check_dir/backend" > "$check_dir/backend.log" 2>&1 &
 backend_pid=$!
 for ((i=0;i<150;i++)); do
  if curl -fsS "http://127.0.0.1:$http_port/healthz" >/dev/null 2>&1; then return; fi
  sleep .2
 done
 echo 'Backend did not start' >&2; return 1
}
export RHYTHM_ML_API="http://127.0.0.1:$http_port/api/v1"
export RHYTHM_ML_DB="$db_name"
start_backend
python3 scripts/ml_smoke.py --state "$check_dir/state.json"
kill "$backend_pid"; wait "$backend_pid"; backend_pid=""
start_backend
python3 scripts/ml_smoke.py --resume --state "$check_dir/state.json"
echo 'PASS: complete ML verification including process restart'

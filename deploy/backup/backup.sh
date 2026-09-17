#!/bin/sh
# Ежедневный бэкап Postgres (ТЗ R.5: не реже 1 раза в сутки).
# Хранит последние $BACKUP_RETENTION_DAYS архивов в /backups (volume backup_data).
set -eu
: "${POSTGRES_USER:?POSTGRES_USER is required}"
: "${POSTGRES_PASSWORD:?POSTGRES_PASSWORD is required}"
: "${POSTGRES_NAME:?POSTGRES_NAME is required}"

DIR=/backups
RETENTION="${BACKUP_RETENTION_DAYS:-7}"
mkdir -p "$DIR"
export PGPASSWORD="$POSTGRES_PASSWORD"

while true; do
  TS=$(date -u +%Y%m%dT%H%M%SZ)
  F="$DIR/rhytm-$TS.sql.gz"
  if pg_dump -h postgres -U "$POSTGRES_USER" -d "$POSTGRES_NAME" | gzip > "$F"; then
    echo "backup ok: $F"
  else
    echo "backup FAILED" >&2
    rm -f "$F"
  fi
  # retention: удалить всё старше последних $RETENTION архивов
  ls -1t "$DIR"/rhytm-*.sql.gz 2>/dev/null | tail -n +"$((RETENTION + 1))" | xargs -r rm -f
  sleep 86400
done

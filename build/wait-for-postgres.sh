#!/bin/bash
set -e

until (echo -n > /dev/tcp/$DB_HOST/5432) >/dev/null 2>&1 ; do
  >&2 echo "Postgres is unavailable - sleeping"
  sleep 1
done

>&2 echo "Postgres is up - executing command"
exec $@

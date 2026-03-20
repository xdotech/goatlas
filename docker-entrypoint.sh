#!/bin/sh
set -e

# Run migrations before starting the server
echo "Running database migrations..."
/app/goatlas migrate

echo "Starting goatlas $*..."
exec gosu goatlas /app/goatlas "$@"

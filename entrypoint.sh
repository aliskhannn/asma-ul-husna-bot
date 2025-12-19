#!/bin/sh
set -e

echo "Running migrations..."
/usr/local/bin/goose -dir ./migrations postgres "$DATABASE_URL" up

echo "Starting bot..."
exec ./husna-bot

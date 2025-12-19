#!/bin/sh
set -e

echo "Running migrations..."
goose -dir ./migrations postgres "$DATABASE_URL" up

echo "Starting bot..."
exec ./husna-bot

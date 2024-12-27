#!/bin/bash

set -o pipefail

# Function to start the PostgreSQL container
start_container() {
  # Check if container is already running
  if docker ps --filter "name=^$PG_CONTAINER_NAME$" --format '{{.Names}}' | grep -q "^$PG_CONTAINER_NAME$"; then
    echo "PostgreSQL container '$PG_CONTAINER_NAME' is already running."
    exit 1
  fi

    # Start the container
  docker run --name "$PG_CONTAINER_NAME" \
      -e POSTGRES_USER="$PG_USER" \
      -e POSTGRES_PASSWORD="$PG_PASSWORD" \
      -e POSTGRES_DB="$DB_NAME" \
      -p "$PG_PORT":5432 \
      -d postgres:17.2
  echo "PostgreSQL container '$PG_CONTAINER_NAME' started."

  # Wait for PostgreSQL to be ready
  TIMEOUT=60
  START_TIME=$(date +%s)
  until docker exec "$PG_CONTAINER_NAME" pg_isready -U "$PG_USER" -d "$DB_NAME" > /dev/null 2>&1; do
    CURRENT_TIME=$(date +%s)
    ELAPSED_TIME=$((CURRENT_TIME - START_TIME))

    if [ $ELAPSED_TIME -ge $TIMEOUT ]; then
      echo "Timeout reached. PostgreSQL is not ready after $TIMEOUT seconds."
      exit 1
    fi

    echo "Waiting for PostgreSQL to be ready..."
    sleep 2
  done

  echo "PostgreSQL is ready."

  # Apply schema if exists
  if [ -f "$SCHEMA_FILE" ] && [ -r "$SCHEMA_FILE" ]; then
    docker cp "$SCHEMA_FILE" "$PG_CONTAINER_NAME":/tmp/schema.sql
    docker exec -u postgres "$PG_CONTAINER_NAME" psql -U "$PG_USER" -d "$DB_NAME" -f /tmp/schema.sql
    echo "Schema applied successfully."
  else
    echo "Schema file '$SCHEMA_FILE' not found or unreadable. Skipping."
  fi
}

# Function to stop the PostgreSQL container
stop_container() {
  # Stop and remove the container if it is running
  if docker ps --filter "name=^$PG_CONTAINER_NAME$" --format '{{.Names}}' | grep -q "^$PG_CONTAINER_NAME$"; then
    docker stop "$PG_CONTAINER_NAME"
    docker rm "$PG_CONTAINER_NAME"
    echo "PostgreSQL container '$PG_CONTAINER_NAME' stopped and removed."
  else
    echo "PostgreSQL container '$PG_CONTAINER_NAME' is not running."
  fi
}

# Function to restart the PostgreSQL container
restart_container() {
  stop_container
  start_container
}

# Main script to handle commands
case "$1" in
  start)
    start_container
    ;;
  stop)
    stop_container
    ;;
  restart)
    restart_container
    ;;
  *)
    echo "Usage: $0 {start|stop|restart}"
    exit 1
    ;;
esac

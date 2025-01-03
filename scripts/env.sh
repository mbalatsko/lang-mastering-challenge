# Environment variables
export PG_CONTAINER_NAME=${PG_CONTAINER_NAME:-lmc_challenge_pg_container}  # Default container name
export PG_HOST=${PG_HOST:-127.0.0.1} # Host for PostgreSQL to run on
export PG_PORT=${PG_PORT:-5432}  # Port for PostgreSQL to listen on
export PG_USER=${PG_USER:-tester}  # PostgreSQL user name
export PG_PASSWORD=${PG_PASSWORD:-tester}  # Password for the PostgreSQL user
export DB_NAME=${DB_NAME:-task_manager}  # Name of the database to create
export SCHEMA_FILE=${SCHEMA_FILE:-"./scripts/database/schema.sql"}  # Path to the SQL schema file

export JWT_SECRET=${SCHEMA_FILE:-"super-mega-secret"} # Secret used to sign JWT tokens

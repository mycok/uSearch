#!/bin/bash
 set -eo pipefail
# Check if both the DB_NAME and DB_HOST argument values have been provided.
if [[ $# -ne 2 ]]; then
    echo "Usage: bootstrap DB_NAME DB_HOST"
    exit 1
fi

DB_NAME=$1
DB_HOST=$2

# Write and subsequently append SQL migration steps to a /tmp/migrations-all.sql
# file.
echo "CREATE DATABASE IF NOT EXISTS $1; USE $1;" > /tmp/migrations-all.sql
cat /migrations/*.up* >> /tmp/migrations-all.sql
cat "\q" >> /tmp/migrations-all.sql

# Apply the schema in a loop so we can wait until the CDB cluster is ready.
until cat /tmp/migrations-all.sql | ./cockroach sql --insecure --echo-sql --host $DB_HOST; do
    sleep 5;
done
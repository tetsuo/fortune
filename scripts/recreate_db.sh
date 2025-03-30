#!/usr/bin/env -S bash -e

# Script for dropping and creating a new database locally using Docker.

docker rm -f mysql
./scripts/docker_mysql.sh

MAX_RETRIES=5
DELAY=5  # Seconds to wait between drop retries
COUNT=0

# It takes approx. 15 seconds for MySQL to be ready.

while [ $COUNT -lt $MAX_RETRIES ]; do
    echo "Attempt $(($COUNT + 1)) of $MAX_RETRIES..."

    if go run ./devtools/cmd/db drop; then
        echo "Database dropped successfully."
        ./scripts/create_local_db.sh
    fi

    echo "Failed. Retrying in $DELAY seconds..."
    COUNT=$((COUNT + 1))
    sleep $DELAY
done

echo "Failed dropping the database after $MAX_RETRIES attempts."
exit 1

#!/usr/bin/env -S bash -e

# Script for dropping all test databases.

for dbname in \
    fortune_mysql_test \
    "fortune_mysql_test_0" \
    "fortune_mysql_test_1" \
    "fortune_mysql_test_2" \
    "fortune_mysql_test_3"; do
    DATABASE_NAME=$dbname LOG_LEVEL=info go run devtools/cmd/db/main.go drop
done

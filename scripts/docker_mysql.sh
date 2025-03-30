#!/usr/bin/env -S bash -e

docker run --name mysql \
  -e MYSQL_ROOT_PASSWORD=example \
  -e MYSQL_DATABASE=fortune_db \
  -p 3306:3306 \
  -d mysql:8.0 \
  --log-bin-trust-function-creators=1 \
  --general-log=1 \
  --general-log-file=/var/lib/mysql/general.log \
  --bind-address=0.0.0.0

# Tail logs: docker exec -it mysql tail -f /var/lib/mysql/general.log

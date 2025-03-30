# Development

# Makefile

The `Makefile` is intended for development only, providing commands for building, testing, formatting, and cleaning the project.

## Build

Compile a production build for your environment:

```
make build
```

Outputs to `bin/`.

## Testing

Run tests with race detection and generate a coverage report:

```
make cover
```

Check function coverage based on `coverage.out`:

```
make cover-func
```

Generate an HTML coverage report:

```
make cover-html
```

## Cleanup

Remove build artifacts and cached files:

```
make clean
```

Delete test coverage files:

```
make testclean
```

Remove all build outputs, including `bin/`, `build/`, and `dist/`:

```
make distclean
```

## Code formatting & dependencies

Check Go formatting:

```
make gofmt
```

Clean up Go module dependencies:

```
make tidy
```

# Linting

The `all.bash` script handles CI tasks such as linting, static analysis, and testing. It helps maintain code quality before commits, pushes, and deployments.

## Usage

Run the script with:

```sh
./all.bash [subcommand]
```

If no subcommand is provided, it runs standard linters and tests.

## Subcommands

- `help` → Displays usage information.
- _(default)_ → Runs all standard checks and tests.
- `build` → Compiles the project using `make build`.
- `ci` → Runs checks and tests in a CI-friendly way.
- `cl` → Runs checks on modified files, for use before committing or pushing.
- `lint` → Runs all linters listed below:
  - `misspell` → Detects spelling errors in source files.
  - `migrations` → Checks for incorrect database migration sequence numbers.
  - `ineffassign` → Detects unused assignments.
  - `errcheck` → Checks for unhandled errors.
  - `staticcheck` → Runs static analysis on the codebase.
  - `unconvert` → Detects unnecessary type conversions.
  - `vet` → Runs `go vet` for additional static analysis.

### When to use

- Before committing, run:

  ```sh
  ./all.bash cl
  ```

  This ensures only modified files are checked.

- Before pushing to CI, run:

  ```sh
  ./all.bash ci
  ```

  This runs the full suite of tests and checks.

- To manually run a specific linter, use:
  ```sh
  ./all.bash staticcheck
  ```

# `scripts/`

These scripts are for local MYSQL setup only.

## Environment variables

They all use these environment variables to configure database access:

```sh
database_user="${DATABASE_USER:-root}"
database_password="${DATABASE_PASSWORD:-example}"
database_host="${DATABASE_HOST:-127.0.0.1}"
database_port="${DATABASE_PORT:-3306}"
database_name="${DATABASE_NAME:-fortune_db}"
```

To override these, export custom values before running scripts:

```sh
export DATABASE_USER=myuser
export DATABASE_PASSWORD=mypassword
export DATABASE_NAME=my_custom_db
...
```

## `create_local_db.sh`

Creates a fresh local database and applies migrations.

### What it does:

1. Runs `go run devtools/cmd/db/main.go create` to create the database.
2. Calls `migrate_db.sh up` to apply all migrations.

## `docker_mysql.sh`

Starts a MySQL 8.0 instance in Docker.

### What it does:

- Runs a MySQL container (`mysql:8.0`) with:
  - Root password: `example`
  - Default database: `fortune_db`
  - Port: `3306`
  - General query logging enabled
  - Function-based logging allowed (`--log-bin-trust-function-creators=1`)

### How to check logs:

```sh
docker exec -it mysql tail -f /var/lib/mysql/general.log
```

## `drop_test_dbs.sh`

Drops all test databases used in automated tests.

### What it does:

Loops through test databases (`fortune_mysql_test`, `fortune_mysql_test_0`, etc.) and deletes them using:

```sh
go run devtools/cmd/db/main.go drop
```

## `migrate_db.sh`

Manages database migrations.

### Usage:

```sh
./scripts/migrate_db.sh [up|down|force|version]
```

### What it does:

- Runs `migrate` using migration files from `etc/migrations/`.

### Available commands:

- `up` → Applies all pending migrations.
- `down` → Rolls back the last applied migration.
- `force` → Forces migration to a specific version.
- `version` → Shows the current migration version.

Use this to apply or rollback schema changes.

## `recreate_db.sh`

Completely resets and recreates the local database.

### What it does:

1. Stops and removes the MySQL container.
2. Starts a new one (`docker_mysql.sh`).
3. Drops the database (`go run ./devtools/cmd/db drop`).
4. Creates a fresh database (`create_local_db.sh`).

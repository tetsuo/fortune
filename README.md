# Fortune API

Fortune is a service for serving fortune cookies. It reuses internal code from [pkgsite](https://github.com/golang/pkgsite) (the Go package index), adapted for MySQL, and implements a lean, GCP-native stack with full observability. The project showcases a production-ready Go setup you can build on or deploy as-is.

- [Installation](#installation)
- [API specification](./API.md)
- [Development guide](./DEVELOPMENT.md)

## Installation

Below are instructions for running the service locally or with **Kind**.

### Prerequisites

Ensure you have the following installed before proceeding:

- [**Go**](https://go.dev/dl/) (≥ v1.23.3)
- [**golang-migrate**](https://github.com/golang-migrate/migrate) (≥ v4.18.2)
- [**Goreleaser**](https://goreleaser.com/) (≥ v2.1.0)
- [**Docker**](https://www.docker.com/get-started) (latest recommended version)
- [**Kind**](https://kind.sigs.k8s.io/docs/user/quick-start/) (≥ v0.27)
- [**kubectl**](https://kubernetes.io/docs/tasks/tools/) (latest recommended version)
- [**Terraform**](https://developer.hashicorp.com/terraform/downloads) (≥ v1.2.9)

### Clone the repository

```sh
git clone git@github.com:tetsuo/fortune.git
cd fortune
git fetch --tags  # Ensure tags are fetched for build versioning
```

### Install Go dependencies

```sh
make tidy
```

### Spin up a local MySQL instance

Run the following script to start a **MySQL 8.0** container with the `fortune_db` database:

```sh
./scripts/docker_mysql.sh
```

This starts MySQL, sets the root password, and enables general query logging.

💡 _You can modify values in this script, but it's recommended to stick to defaults for simplicity._

### Verify database creation

Check if `fortune_db` exists:

```sh
docker exec -it mysql mysql -u root -p -e "SHOW DATABASES LIKE 'fortune_db';"
```

## Running migrations

Run `migrate_db.sh` to apply or manage database migrations.

### Check migration status

Ensure **go-migrate** is installed, then verify the current migration version:

```sh
./scripts/migrate_db.sh version
```

💡 If you see `error: no migration`, it means no migrations have been applied yet.

### Apply migrations

Run the MySQL migrations from [etc/migrations](./etc/migrations/):

```sh
./scripts/migrate_db.sh up
```

### Verify table creation

```sh
docker exec -it mysql \
  mysql -u root -p -e "USE fortune_db; SHOW TABLES LIKE 'fortune_cookies';"
```

## Building & running the application

### Build the application

```sh
make build
```

### Start the fortune server

```sh
LOG_LEVEL=debug ./bin/frontend
```

(Or run with `go run ./cmd/frontend/...`.)

If everything works, you should see logs like this:

```
2025-03-14T13:24:43.766+0100    INFO    frontend/main.go:116    debug server listening on localhost:8081
2025-03-14T13:24:43.772+0100    INFO    frontend/main.go:192    frontend server listening on localhost:8080
```

## Fetching your first fortune cookie

Try retrieving a fortune cookie:

```sh
curl localhost:8080 ; echo
```

🚨 **Expected output:** `Not Found` (because we haven't added fortunes yet).

### Upload fortunes

Bulk insert a `fortunes.txt` file containing **2,000+ fortunes**:

```sh
curl -X POST -H "Content-Type: text/plain" \
  --data-binary @fortunes.txt \
  http://localhost:8080 -v
```

### Retrieve a fortune

```sh
curl localhost:8080 ; echo
```

🔮 **Example output:** `"One planet is all you get."`

## Explore the local debug server

Visit [localhost:8081](http://localhost:8081) for debugging insights, metrics, and other useful details.

## Running tests

Before deploying, ensure all tests pass:

```sh
./all.bash ci
```

This installs linters and runs tests. If everything is ✅, continue to release preparation.

## Releasing

**Goreleaser** handles packaging and release generation.

### Generate a release tarball

```sh
goreleaser release -f .goreleaser.yml --snapshot --clean
```

🚀 This outputs the build to the `dist/` folder.

Quick build (no release):

```sh
goreleaser build -f .goreleaser.yml --snapshot --clean --single-target
```

### Build a Docker image

```sh
docker build -t fortune-frontend:latest .
```

## Deploying to Kind

### Create a Kind cluster

```sh
kind create cluster
```

### Load the Docker image into the cluster

```sh
kind load docker-image fortune-frontend:latest
```

### Set up Kubernetes config

From the root directory:

```sh
kind get kubeconfig --name kind > terraform/kubeconfig.yaml
```

### Initialize Terraform

Navigate to `terraform/` and run:

```sh
terraform init
```

### Apply Terraform changes

```sh
terraform apply
```

## Running migrations on Kubernetes

### Forward MySQL port

```sh
kubectl port-forward -n fortune services/mysql 3306:3306
```

### Run migrations

```sh
DATABASE_USER=kinduser DATABASE_PASSWORD=kindpassword ./scripts/migrate_db.sh up
```

🚨 _In production, run migrations securely. The included mysql chart is **not** meant for production use._

## Exposing the service

### Forward HAProxy port

```sh
kubectl port-forward --namespace ingress-controller service/haproxy-kubernetes-ingress 8080:80
```

### Update your hosts file

Update your `/etc/hosts` file, ensure it contains:

```
127.0.0.1 local.haproxy.kind
127.0.0.1 www.local.haproxy.kind
```

### Upload fortunes again

```sh
curl -X POST -H "Content-Type: text/plain" \
  --data-binary @fortunes.txt \
  http://local.haproxy.kind:8080 -v
```

### 🔮 Get a fortune

```sh
curl local.haproxy.kind:8080 ; echo
```

🎉 **I hope it's a good one!**

## License

MIT license

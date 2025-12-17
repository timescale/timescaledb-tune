# Integration Testing

## Overview

Tests timescaledb-tune against PostgreSQL 10-18 using TimescaleDB Docker containers.

## Workflow (`.github/workflows/integration-tests.yml`)

**Build job** (once):
- Builds binary, caches based on source hash
- Uploads artifact for test jobs

**Test jobs** (9 parallel, PG 10-18):
1. Start TimescaleDB container
2. Run `timescaledb-tune --yes --memory=256MB --cpus=1`
3. Show config diff
4. Restart PostgreSQL
5. Check logs for errors (filtering benign shutdown messages)

## Test Configuration

```bash
docker run -d timescale/timescaledb:latest-pg{version}
docker exec /tmp/timescaledb-tune --yes --memory="256MB" --cpus=1
```

**Note**: PostgreSQL 17/18 may show no config changes. This is expected - their defaults are already within 5% of recommendations for 256MB environments (tool's fudgeFactor tolerance).

## Error Detection

Filters benign messages:
- `received shutdown request` - Normal shutdown
- `terminating` - Process termination
- `due to administrator command` - Shutdown commands
- `background worker` - Worker lifecycle
- `exited with exit code` - Normal exits

## Triggers

- Push to `main`
- All pull requests
- Manual (`workflow_dispatch`)

## Local Testing

```bash
make build
docker run -d --name pg-test -e POSTGRES_PASSWORD=password \
  timescale/timescaledb:latest-pg18

# Get config path
CONFIG_PATH=$(docker exec pg-test psql -U postgres -t -c "SHOW config_file" | xargs)

# Copy and run
docker cp build/timescaledb-tune pg-test:/tmp/timescaledb-tune
docker exec -u root -e TMPDIR=/var/lib/postgresql/data pg-test \
  /tmp/timescaledb-tune --conf-path="${CONFIG_PATH}" --pg-version=18 --yes

# Restart and check
docker restart pg-test && sleep 3
docker logs pg-test 2>&1 | grep -i ERROR

# Cleanup
docker rm -f pg-test
```

## Common Issues

**Permission denied on backup**: Set `TMPDIR=/var/lib/postgresql/data` and use `-u root`

**Grep exit code failure**: Pipeline uses `|| true` to handle no matches

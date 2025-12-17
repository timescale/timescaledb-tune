# Development Guide

## Project Overview

Go CLI tool that tunes PostgreSQL configuration for optimal TimescaleDB performance.

- **Language**: Go 1.23+
- **Version**: 0.18.1 (`pkg/tstune/tuner.go`)

## Developer Preferences

@~/.claude/tigerdata.md

## Quick Start

```bash
# Build
go build -v ./cmd/timescaledb-tune

# Test
go test -v ./...

# Run
./timescaledb-tune --help
```

## Key Files

- `cmd/timescaledb-tune/main.go` - Entry point
- `pkg/pgtune/` - PostgreSQL tuning logic
- `pkg/tstune/` - TimescaleDB orchestration
- `pkg/tstune/utils.go` - ValidPGVersions (10-18)

## Important Behaviors

**Settings comparison**: Uses 5% tolerance (`fudgeFactor = 0.05` in `tuner.go:66`). Settings within 5% of recommendation are not changed.

**Config detection**: Platform-specific paths for `postgresql.conf`. Override with `--conf-path`.

**Mocking in tests**: Uses function variables (`filepathAbsFn`, `getPGConfigVersionFn`, `osStatFn`, `execFn`).

## CI/CD

- **Unit tests**: `.github/workflows/go.yml`
- **Integration tests**: `.github/workflows/integration-tests.yml` - Tests PG 10-18 in parallel

## Documentation

- [Architecture](.claude/docs/ARCHITECTURE.md) - System design
- [Integration Testing](.claude/docs/INTEGRATION_TESTING.md) - Test details
- [README.md](README.md) - User documentation

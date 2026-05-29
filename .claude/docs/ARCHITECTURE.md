# Architecture

## Project Structure

```
cmd/timescaledb-tune/    # Entry point
pkg/pgtune/              # PostgreSQL tuning (memory, parallel, wal, misc)
pkg/tstune/              # Orchestration (tuner, config detection, backup)
pkg/pgutils/             # Version utilities
internal/parse/          # Byte/time parsing
```

## Core Interfaces

**Recommender** (`pkg/pgtune/tune.go`)
- `IsAvailable()` - Can recommender run?
- `Recommend(key)` - Get PostgreSQL-formatted value

**SettingsGroup** (`pkg/pgtune/tune.go`)
- `Label()` - Group name
- `Keys()` - Parameter names
- `GetRecommender(profile)` - Get recommender

**SystemConfig** (`pkg/pgtune/tune.go`)
- Wraps system resources (memory, CPUs, PG version)

**Tuner** (`pkg/tstune/tuner.go`)
- Main orchestrator, `Run()` executes workflow

## Settings Groups

- **Memory**: shared_buffers (25% memory), effective_cache_size (75%), maintenance_work_mem, work_mem
- **Parallelism**: max_worker_processes, max_parallel_workers_per_gather, max_parallel_workers, timescaledb.max_background_workers
- **WAL**: wal_buffers, min_wal_size, max_wal_size, checkpoint_timeout, wal_compression
- **Misc**: max_connections, random_page_cost (1.1), effective_io_concurrency (200), max_locks_per_transaction, autovacuum settings

## Workflow

1. Init - Parse flags, detect config, get PG version
2. Restore mode - If `--restore`, handle and exit
3. Check shared_preload_libraries has timescaledb
4. Tune settings - For each group: compare current vs recommended (5% tolerance), prompt/apply, backup, write
5. Output - Show success, backup location

## Parsing

**Bytes** (`internal/parse/parse.go`): TB, GB, MB, kB (powers of 1024)
**Time** (`internal/parse/time.go`): us, ms, s, min, h, d

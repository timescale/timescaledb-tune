## timescaledb-tune

`timescaledb-tune` is a program for tuning a
[TimescaleDB](//github.com/timescale/timescaledb) database to perform
its best based on the host's resources such as memory and number of CPUs.
It parses the existing `postgresql.conf` file to ensure that the TimescaleDB
extension is appropriately installed and provides recommendations for
memory, parallelism, WAL, and other settings.

### Getting started
You need the Go runtime (1.12+) installed, then simply `go install` this repo:
```bash
$ go install github.com/timescale/timescaledb-tune/cmd/timescaledb-tune@main
```

It is also available as a binary package on a variety systems using
Homebrew, `yum`, or `apt`. Search for `timescaledb-tools`.

### Using timescaledb-tune
By default, `timescaledb-tune` attempts to locate your `postgresql.conf`
file for parsing by using heuristics based on the operating system, so the
simplest invocation would be:
```bash
$ timescaledb-tune
```

You'll then be given a series of prompts that require minimal user input to
make sure your config file is up to date:
```text
Using postgresql.conf at this path:
/usr/local/var/postgres/postgresql.conf

Is this correct? [(y)es/(n)o]: y
Writing backup to:
/var/folders/cr/zpgdkv194vz1g5smxl_5tggm0000gn/T/timescaledb_tune.backup201901071520

shared_preload_libraries needs to be updated
Current:
#shared_preload_libraries = 'timescaledb'
Recommended:
shared_preload_libraries = 'timescaledb'
Is this okay? [(y)es/(n)o]: y
success: shared_preload_libraries will be updated

Tune memory/parallelism/WAL and other settings? [(y)es/(n)o]: y
Recommendations based on 8.00 GB of available memory and 4 CPUs for PostgreSQL 11

Memory settings recommendations
Current:
shared_buffers = 128MB
#effective_cache_size = 4GB
#maintenance_work_mem = 64MB
#work_mem = 4MB
Recommended:
shared_buffers = 2GB
effective_cache_size = 6GB
maintenance_work_mem = 1GB
work_mem = 26214kB
Is this okay? [(y)es/(s)kip/(q)uit]:
```

If you have moved the configuration file to a different location, or
auto-detection fails (file an issue please!), you can provide the location
with the `--conf-path` flag:
```bash
$ timescaledb-tune --conf-path=/path/to/postgresql.conf
```

At the end, your `postgresql.conf` will be overwritten with the changes
that you accepted from the prompts.

#### Other invocations

By default, timescaledb-tune provides recommendations for a typical timescaledb workload. The `--profile` flag can be
used to tailor the recommendations for other workload types. Currently, the only non-default profile is "promscale".
The `TSTUNE_PROFILE` environment variable can also be used to affect this behavior.

```bash
$ timescaledb-tune --profile promscale
```

If you want recommendations for a specific amount of memory and/or CPUs:
```bash
$ timescaledb-tune --memory="4GB" --cpus=2
```

If you want to set a specific number of background workers (`timescaledb.max_background_workers`):
```bash
$ timescaledb-tune --max-bg-workers=16
```

If you have a dedicated disk for WAL, or want to specify how much of a
shared disk should be used for WAL:
```bash
$ timescaledb-tune --wal-disk-size="10GB"
```

If you want to accept all recommendations, you can use `--yes`:
```bash
$ timescaledb-tune --yes
```

If you just want to see the recommendations without writing:
```bash
$ timescaledb-tune --dry-run
```

If there are too many prompts:
```bash
$ timescaledb-tune --quiet
```

And if you want to skip all prompts and get quiet output:
```bash
$ timescaledb-tune --quiet --yes
```

And if you want to append the recommendations to the end of your conf file
instead of in-place replacement:
```bash
$ timescaledb-tune --quiet --yes --dry-run >> /path/to/postgresql.conf
```

### Restoring backups

`timescaledb-tune` makes a backup of your `postgresql.conf` file each time
it runs (without the `--dry-run` flag) in your temp directory. If you find
that the configuration given is not working well, you can restore a backup
by using the `--restore` flag:
```bash
$ timescaledb-tune --restore
```
```text
Using postgresql.conf at this path:
/usr/local/var/postgres/postgresql.conf

Is this correct? [(y)es/(n)o]: y
Available backups (most recent first):
1) timescaledb_tune.backup201901222056 (14 hours ago)
2) timescaledb_tune.backup201901221640 (18 hours ago)
3) timescaledb_tune.backup201901221050 (24 hours ago)
4) timescaledb_tune.backup201901211817 (41 hours ago)

Use which backup? Number or (q)uit: 1
Restoring 'timescaledb_tune.backup201901222056'...
success: restored successfully
```

### Contributing
We welcome contributions to this utility, which like TimescaleDB is
released under the Apache2 Open Source License.  The same [Contributors Agreement](//github.com/timescale/timescaledb/blob/master/CONTRIBUTING.md)
applies; please sign the [Contributor License Agreement](https://cla-assistant.io/timescale/timescaledb-tune) (CLA) if you're a new contributor.

### Releasing

Please follow the instructions [here](https://github.com/timescale/release-build-scripts/blob/release_tools/README.md#publishing-changes-to-timescaledb-tune) 
to publish a release of this tool.

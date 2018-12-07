## timescaledb-tune

`timescaledb-tune` is a program for tuning a
[TimescaleDB](//github.com/timescale/timescaledb) database to perform
its best based on the host's resources such as memory and number of CPUs.
It parses the existing `postgresql.conf` file to ensure that the TimescaleDB
extension is appropriately installed and provides recommendations for memory,
parallelism, WAL, and other settings.

### Getting started
You need the Go runtime (1.10+) installed, then simply `go get` this repo:
```bash
$ go get github.com/timescale/timescaledb-tune/cmd/timescaledb-tune
```

### Using timescaledb-tune
By default, `timescaledb-tune` attempts to locate your `postgresql.conf` file
for parsing by using heuristics based on the operating system, so the simplest
invocation would be:
```bash
$ timescaledb-tune
```

You'll then be given a series of prompts that require minimal user input to
make sure your config file is up to date:
```text
Using postgresql.conf at this path:
/usr/local/var/postgres/postgresql.conf

Is this the correct path? [(y)es/(n)o]: y
shared_preload_libraries needs to be updated
Current:
#shared_preload_libraries = 'timescaledb'		# (change requires restart)
Recommended:
shared_preload_libraries = 'timescaledb'		# (change requires restart)
Is this okay? [(y)es/(n)o]: y
success: shared_preload_libraries will be updated

Tune memory/parallelism/WAL and other settings?[(y)es/(n)o]: y
Recommendations based on 8.00 GB of available memory and 4 CPUs for PostgreSQL 10

Memory settings recommendations
Current:
shared_buffers = 128MB			# min 128kB
#effective_cache_size = 4GB
#maintenance_work_mem = 64MB		# min 1MB
#work_mem = 4MB				# min 64kB
Recommended:
shared_buffers = 2GB			# min 128kB
effective_cache_size = 6GB
maintenance_work_mem = 1GB		# min 1MB
work_mem = 26214kB				# min 64kB
Is this okay? [(y)es/(s)kip/(q)uit]:
```

If you have moved the configuration file to a different location, or
auto-detection fails (file an issue please!), you can provide the location with
the `--conf-path` flag:
```bash
$ timescaledb-tune --conf-path=/path/to/postgresql.conf
```

At the end, your `postgresql.conf` will be overwritten with the changes that you
accepted from the prompts.

#### Other invocations

If you want recommendations for a specific amount of memory and/or CPUs:
```bash
$ timescaledb-tune --memory="4GB" --cpus=2
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
$ timescaledb-tune --quiet --yes --dry-run > /path/to/postgresql.conf
```

### Contributing
We welcome contributions to this utility, which like TimescaleDB is released under the Apache2 Open Source License.  The same [Contributors Agreement](//github.com/timescale/timescaledb/blob/master/CONTRIBUTING.md) applies; please sign the [Contributor License Agreement](https://cla-assistant.io/timescale/timescaledb-tune) (CLA) if you're a new contributor.

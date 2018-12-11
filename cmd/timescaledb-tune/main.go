// timescaledb-tune analyzes a user's postgresql.conf file to make sure it is
// ready and tuned to use TimescaleDB. It checks that the TimescaleDB library
// ('timescaledb') is properly listed as a shared preload library and analyzes
// various groups of settings to make sure they are reasonably set for the
// machine's resources.
//
// The groups of settings deal with memory usage, parallelism, the WAL, and
// other miscellaneous settings that have been found to be useful when tuning.
package main

import (
	"flag"
	"fmt"
	"os"
	"runtime"

	"github.com/timescale/timescaledb-tune/pkg/tstune"
)

const version = "0.1.0"

var (
	f           tstune.TunerFlags
	showVersion bool
)

// Parse args
func init() {
	flag.StringVar(&f.Memory, "memory", "", "Amount of memory to base recommendations on in the PostgreSQL format <int value><units>, e.g., 4GB. Default is to use all memory")
	flag.UintVar(&f.NumCPUs, "cpus", 0, "Number of CPU cores to base recommendations on. Default is equal to number of cores")
	flag.StringVar(&f.ConfPath, "conf-path", "", "Path to postgresql.conf. If blank, heuristics will be used to find it")
	flag.StringVar(&f.DestPath, "out-path", "", "Path to write the new configuration file. If blank, will use the same file that is read from")
	flag.StringVar(&f.PGConfig, "pg-config", "pg_config", "Path to the pg_config binary")
	flag.BoolVar(&f.YesAlways, "yes", false, "Answer 'yes' to every prompt")
	flag.BoolVar(&f.Quiet, "quiet", false, "Show only the total recommendations at the end")
	flag.BoolVar(&f.UseColor, "color", true, "Use color in output (works best on dark terminals)")
	flag.BoolVar(&f.DryRun, "dry-run", false, "Whether to just show the changes without overwriting the configuration file")

	flag.BoolVar(&showVersion, "version", false, "Show the version of this tool")
	flag.Parse()
}

func main() {
	if showVersion {
		fmt.Printf("%s (%s %s)\n", version, runtime.GOOS, runtime.GOARCH)
	} else {
		tuner := tstune.Tuner{}
		tuner.Run(&f, os.Stdin, os.Stdout, os.Stderr)
	}
}

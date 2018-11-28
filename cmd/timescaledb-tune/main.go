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
	"os"

	"github.com/timescale/timescaledb-tune/pkg/tstune"
)

var f tstune.TunerFlags

// Parse args
func init() {
	flag.StringVar(&f.ConfPath, "conf-path", "", "Path to postgresql.conf. If blank, heuristics will be used to find it")
	flag.StringVar(&f.DestPath, "out-path", "", "Path to write the new configuration file. If blank, will use the same file that is read from")
	flag.BoolVar(&f.YesAlways, "yes", false, "Answer 'yes' to every prompt")
	flag.BoolVar(&f.Quiet, "quiet", false, "Show only the total recommendations at the end")
	flag.BoolVar(&f.UseColor, "color", true, "Use color in output (works best on dark terminals)")
	flag.BoolVar(&f.DryRun, "dry-run", false, "Whether to just show the changes without overwriting the configuration file")
	flag.Parse()
}

func main() {
	tuner := tstune.Tuner{}
	tuner.Run(&f, os.Stdin, os.Stdout, os.Stderr)
}

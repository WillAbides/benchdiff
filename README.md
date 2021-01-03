# benchdiff

[![godoc](https://godoc.org/github.com/willabides/benchdiff?status.svg)](https://godoc.org/github.com/willabides/benchdiff)
[![ci](https://github.com/WillAbides/benchdiff/workflows/ci/badge.svg?branch=main&event=push)](https://github.com/WillAbides/benchdiff/actions?query=workflow%3Aci+branch%3Amaster+event%3Apush)

Benchdiff is a command line tool intended to help speed up the feedback loop for go benchmarks.

The old workflow:
- `go test -bench MyBenchmark -run '^$' -count 10 . > tmp/bench.out`
- `git stash && git switch main`
- `go test -bench MyBenchmark -run '^$' -count 10 . > tmp/bench-main.out`
- `git switch - && git stash apply`
- `benchstat tmp/bench-main.out tmp/bench.out`

The new workflow:
- `benchdiff --bench 'MyBenchmark'`

## Usage

<!--- start usage output --->
```
Usage: benchdiff

benchdiff runs go benchmarks on your current git worktree and a base ref then uses benchstat to show
the delta.

More documentation at https://github.com/willabides/benchdiff.

Flags:
  -h, --help       Show context-sensitive help.
      --version    Output the benchdiff version and exit.

      --base-ref="HEAD"    The git ref to be used as a baseline.
      --cooldown=100ms     How long to pause for cooldown between head and base runs.
      --force-base         Rerun benchmarks on the base reference even if the output already exists.
      --git-cmd="git"      The executable to use for git commands.
      --json               Format output as JSON.
      --on-degrade=0       Exit code when there is a statistically significant degradation in the
                           results.
      --tolerance=10.0     The minimum percent change before a result is considered degraded.

  benchmark command line:
      --bench="."              Run only those benchmarks matching a regular expression. To run all
                               benchmarks, use '--bench .'.
      --benchmark-args=args    Override the default args to the go command. This may be a template.
                               See https://github.com/willabides/benchdiff for details."
      --benchmark-cmd="go"     The command to use for benchmarks.
      --benchmem               Memory allocation statistics for benchmarks.
      --benchtime=DURATION     Run enough iterations of each benchmark to take t, specified as a
                               time.Duration (for example, --benchtime 1h30s). The default is 1
                               second (1s). The special syntax Nx means to run the benchmark N times
                               (for example, -benchtime 100x).
      --count=10               Run each benchmark n times. If --cpu is set, run n times for each
                               GOMAXPROCS value.'
      --cpu=GOMAXPROCS,...     Specify a list of GOMAXPROCS values for which the benchmarks should
                               be executed. The default is the current value of GOMAXPROCS.
      --packages="./..."       Run benchmarks in these packages.
      --show-bench-cmdline     Instead of running benchmarks, output the command that would be used
                               and exit.

  benchstat options:
      --alpha=0.05                 consider change significant if p < α
      --benchstat-output="text"    format for benchstat output (csv,html,markdown or text)
      --delta-test="utest"         significance test to apply to delta: utest, ttest, or none
      --geomean                    print the geometric mean of each file
      --norange                    suppress range columns (CSV and markdown only)
      --reverse-sort               reverse sort order
      --sort="none"                sort by order: delta, name, none
      --split="pkg,goos,goarch"    split benchmarks by labels

  benchmark result cache:
      --cache-dir=STRING    Override the default directory where benchmark output is kept.
      --clear-cache         Remove benchdiff files from the cache dir.
      --show-cache-dir      Output the cache dir and exit.
```
<!--- end usage output --->

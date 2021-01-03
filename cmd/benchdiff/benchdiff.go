package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"github.com/alecthomas/kong"
	"github.com/willabides/benchdiff/cmd/benchdiff/internal"
	"github.com/willabides/benchdiff/cmd/benchdiff/internal/helpprinter"
	"github.com/willabides/benchdiff/pkg/benchstatter"
	"golang.org/x/perf/benchstat"
)

const defaultBenchArgsTmpl = `test -bench {{.Bench}} -run '^$' -benchtime {{.Benchtime}} -benchmem -count {{.Count}} {{.Packages}}`

var benchstatVars = kong.Vars{
	"AlphaDefault":     "0.05",
	"AlphaHelp":        `consider change significant if p < α`,
	"CSVHelp":          `format benchstat output as CSV`,
	"DeltaTestHelp":    `significance test to apply to delta: utest, ttest, or none`,
	"DeltaTestDefault": `utest`,
	"DeltaTestEnum":    `utest,ttest,none`,
	"GeomeanHelp":      `print the geometric mean of each file`,
	"HTMLHelp":         `format benchstat output as an HTML table`,
	"MarkdownHelp":     `format benchstat output as markdown tables`,
	"NorangeHelp":      `suppress range columns (CSV and markdown only)`,
	"ReverseSortHelp":  `reverse sort order`,
	"SortHelp":         `sort by order: delta, name, none`,
	"SortEnum":         `delta,name,none`,
	"SplitHelp":        `split benchmarks by labels`,
	"SplitDefault":     `pkg,goos,goarch`,
}

type benchstatOpts struct {
	Alpha       float64 `kong:"default=${AlphaDefault},help=${AlphaHelp},group=benchstat"`
	CSV         bool    `kong:"help=${CSVHelp},xor='outputformat',group=benchstat"`
	DeltaTest   string  `kong:"help=${DeltaTestHelp},default=${DeltaTestDefault},enum='utest,ttest,none',group=benchstat"`
	Geomean     bool    `kong:"help=${GeomeanHelp},group=benchstat"`
	HTML        bool    `kong:"help=${HTMLHelp},xor='outputformat',group=benchstat"`
	Markdown    bool    `kong:"help=${MarkdownHelp},group=benchstat"`
	Norange     bool    `kong:"help=${NorangeHelp},group=benchstat"`
	ReverseSort bool    `kong:"help=${ReverseSortHelp},group=benchstat"`
	Sort        string  `kong:"help=${SortHelp},enum=${SortEnum},default=none,group=benchstat"`
	Split       string  `kong:"help=${SplitHelp},default=${SplitDefault},group=benchstat"`
}

var version string

var benchVars = kong.Vars{
	"version":         version,
	"BenchCmdDefault": `go`,
	"CacheDirDefault": filepath.FromSlash("./tmp"),
	"BenchCountHelp":  `Run each benchmark n times.`,
	"BenchHelp":       `Run only those benchmarks matching a regular expression.`,
	"GoArgsHelp":      `Override the default args to the go command. This may be a template. See https://github.com/willabides/benchdiff for the default value."`,
	"BenchtimeHelp":   `The -benchtime argument for the go test command`,
	"PackagesHelp":    `Run benchmarks in these packages.`,
	"BenchCmdHelp":    `The go command to use for benchmarks.`,
	"CacheDirHelp":    `The directory where benchmark output will kept between runs.`,
	"BaseRefHelp":     `The git ref to be used as a baseline.`,
	"CooldownHelp":    `How long to pause for cooldown between head and base runs.`,
	"ForceBaseHelp":   `Rerun benchmarks on the base reference even if the output already exists.`,
	"OnDegradeHelp":   `Exit code when there is a statistically significant degradation in the results.`,
	"JSONOutputHelp":  `Format output as JSON. When true the --csv and --html flags affect only the "benchstat_output" field.`,
	"GitCmdHelp":      `The executable to use for git commands.`,
	"ToleranceHelp":   `The minimum percent change before a result is considered degraded.`,
	"VersionHelp":     `Output the benchdiff version and exit.`,
}

var groupHelp = kong.Vars{
	"benchstatGroupHelp": "benchstat output options:",
	"gotestGroupHelp":    "'go test' options:",
}

var cli struct {
	Version kong.VersionFlag `kong:"help=${VersionHelp}"`

	BaseRef    string        `kong:"default=HEAD,help=${BaseRefHelp},group='x'"`
	Cooldown   time.Duration `kong:"default='100ms',help=${CooldownHelp},group='x'"`
	CacheDir   string        `kong:"type=dir,default=${CacheDirDefault},help=${CacheDirHelp},group='x'"`
	ForceBase  bool          `kong:"help=${ForceBaseHelp},group='x'"`
	GitCmd     string        `kong:"default=git,help=${GitCmdHelp},group='x'"`
	JSONOutput bool          `kong:"help=${JSONOutputHelp},group='x'"`
	OnDegrade  int           `kong:"name=on-degrade,default=0,help=${OnDegradeHelp},group='x'"`
	Tolerance  float64       `kong:"default='10.0',help=${ToleranceHelp},group='x'"`

	Bench     string `kong:"default='.',help=${BenchHelp},group='gotest'"`
	Benchtime string `kong:"default='1s',help=${BenchtimeHelp},group='gotest'"`
	Count     int    `kong:"default=10,help=${BenchCountHelp},group='gotest'"`
	GoArgs    string `kong:"placeholder='args',help=${GoArgsHelp},group='gotest'"`
	GoCmd     string `kong:"default=${BenchCmdDefault},help=${BenchCmdHelp},group='gotest'"`
	Packages  string `kong:"default='./...',help=${PackagesHelp},group='gotest'"`

	BenchstatOpts benchstatOpts `kong:"embed"`
}

const description = `
benchdiff runs go benchmarks on your current git worktree and a base ref then
uses benchstat to show the delta.

See https://github.com/willabides/benchdiff for more details.
`

func main() {
	kctx := kong.Parse(&cli, benchstatVars, benchVars, groupHelp,
		kong.Help(helpprinter.NewHelpPrinter(nil)),
		kong.Description(strings.TrimSpace(description)),
	)
	if cli.GoArgs == "" {
		cli.GoArgs = defaultBenchArgsTmpl
	}
	tmpl, err := template.New("").Parse(cli.GoArgs)
	kctx.FatalIfErrorf(err)
	var benchArgs bytes.Buffer
	err = tmpl.Execute(&benchArgs, cli)
	kctx.FatalIfErrorf(err)

	bd := &internal.Benchdiff{
		BenchCmd:   cli.GoCmd,
		BenchArgs:  benchArgs.String(),
		ResultsDir: cli.CacheDir,
		BaseRef:    cli.BaseRef,
		Path:       ".",
		Writer:     os.Stdout,
		Benchstat:  buildBenchstat(cli.BenchstatOpts),
		Force:      cli.ForceBase,
		GitCmd:     cli.GitCmd,
		BasePause:  cli.Cooldown,
	}
	result, err := bd.Run()
	kctx.FatalIfErrorf(err)

	outputFormat := "human"
	if cli.JSONOutput {
		outputFormat = "json"
	}

	err = result.WriteOutput(os.Stdout, &internal.RunResultOutputOptions{
		BenchstatFormatter: buildBenchstat(cli.BenchstatOpts).OutputFormatter,
		OutputFormat:       outputFormat,
		Tolerance:          cli.Tolerance,
	})
	kctx.FatalIfErrorf(err)
	if result.HasDegradedResult(cli.Tolerance) {
		os.Exit(cli.OnDegrade)
	}
}

var deltaTestOpts = map[string]benchstat.DeltaTest{
	"none":  benchstat.NoDeltaTest,
	"utest": benchstat.UTest,
	"ttest": benchstat.TTest,
}

var sortOpts = map[string]benchstat.Order{
	"none":  nil,
	"name":  benchstat.ByName,
	"delta": benchstat.ByDelta,
}

func buildBenchstat(opts benchstatOpts) *benchstatter.Benchstat {
	order := sortOpts[opts.Sort]
	reverse := opts.ReverseSort
	if order == nil {
		reverse = false
	}
	formatter := benchstatter.TextFormatter(nil)
	if opts.CSV {
		formatter = benchstatter.CSVFormatter(&benchstatter.CSVFormatterOptions{
			NoRange: opts.Norange,
		})
	}
	if opts.HTML {
		formatter = benchstatter.HTMLFormatter(nil)
	}
	if opts.Markdown {
		formatter = benchstatter.MarkdownFormatter(&benchstatter.MarkdownFormatterOptions{
			CSVFormatterOptions: benchstatter.CSVFormatterOptions{
				NoRange: opts.Norange,
			},
		})
	}

	return &benchstatter.Benchstat{
		DeltaTest:       deltaTestOpts[opts.DeltaTest],
		Alpha:           opts.Alpha,
		AddGeoMean:      opts.Geomean,
		SplitBy:         strings.Split(opts.Split, ","),
		Order:           order,
		ReverseOrder:    reverse,
		OutputFormatter: formatter,
	}
}

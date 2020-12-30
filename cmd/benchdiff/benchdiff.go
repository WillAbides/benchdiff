package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/alecthomas/kong"
	"github.com/willabides/benchdiff/cmd/benchdiff/internal"
	"github.com/willabides/benchdiff/pkg/benchstatter"
	"golang.org/x/perf/benchstat"
)

const defaultBenchArgsTmpl = `test -bench {{.Bench}} -run '^$' -benchtime {{.Benchtime}} -benchmem -count {{.BenchCount}} {{.Packages}}`

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
	Alpha       float64 `kong:"default=${AlphaDefault},help=${AlphaHelp}"`
	CSV         bool    `kong:"help=${CSVHelp},xor='outputformat'"`
	DeltaTest   string  `kong:"help=${DeltaTestHelp},default=${DeltaTestDefault},enum='utest,ttest,none'"`
	Geomean     bool    `kong:"help=${GeomeanHelp}"`
	HTML        bool    `kong:"help=${HTMLHelp},xor='outputformat'"`
	Markdown    bool    `kong:"help=${MarkdownHelp}"`
	Norange     bool    `kong:"help=${NorangeHelp}"`
	ReverseSort bool    `kong:"help=${ReverseSortHelp}"`
	Sort        string  `kong:"help=${SortHelp},enum=${SortEnum},default=none"`
	Split       string  `kong:"help=${SplitHelp},default=${SplitDefault}"`
}

var version string

var benchVars = kong.Vars{
	"version":          version,
	"BenchCmdDefault":  `go`,
	"BenchArgsDefault": defaultBenchArgsTmpl,
	"CacheDirDefault":  filepath.FromSlash("./tmp"),
	"BenchCountHelp":   `Run each benchmark n times.`,
	"BenchHelp":        `Run only those benchmarks matching a regular expression.`,
	"BenchArgsHelp":    `Use these arguments to run benchmarks. It may be a template.`,
	"BenchtimeHelp":    `The -benchtime argument for the go test command`,
	"PackagesHelp":     `Run benchmarks in these packages.`,
	"BenchCmdHelp":     `The go command to use for benchmarks.`,
	"CacheDirHelp":     `The directory where benchmark output will kept between runs.`,
	"BaseRefHelp":      `The git ref to be used as a baseline.`,
	"ForceBaseHelp":    `Rerun benchmarks on the base reference even if the output already exists.`,
	"OnDegradeHelp":    `Exit code when there is a statistically significant degradation in the results.`,
	"JSONOutputHelp":   `Format output as JSON. When true the --csv and --html flags affect only the "benchstat_output" field.`,
	"GitCmdHelp":       `The executable to use for git commands.`,
	"VersionHelp":      `Output the benchdiff version and exit.`,
}

var cli struct {
	BaseRef       string           `kong:"default=HEAD,help=${BaseRefHelp}"`
	Bench         string           `kong:"default='.',help=${BenchHelp}"`
	BenchArgs     string           `kong:"default=${BenchArgsDefault},help=${BenchArgsHelp}"`
	BenchCmd      string           `kong:"default=${BenchCmdDefault},help=${BenchCmdHelp}"`
	BenchCount    int              `kong:"default=10,help=${BenchCountHelp}"`
	Benchtime     string           `kong:"default='1s',help=${BenchtimeHelp}"`
	CacheDir      string           `kong:"type=dir,default=${CacheDirDefault},help=${CacheDirHelp}"`
	ForceBase     bool             `kong:"help=${ForceBaseHelp}"`
	GitCmd        string           `kong:"default=git,help=${GitCmdHelp}"`
	JSONOutput    bool             `kong:"help=${JSONOutputHelp}"`
	OnDegrade     int              `kong:"name=on-degrade,default=0,help=${OnDegradeHelp}"`
	Packages      string           `kong:"default='./...',help=${PackagesHelp}"`
	BenchstatOpts benchstatOpts    `kong:"embed"`
	Version       kong.VersionFlag `kong:"help=${VersionHelp}"`
}

const description = `
benchdiff runs go benchmarks on your current git worktree and a base ref then
uses benchstat to show the delta.

See https://github.com/willabides/benchdiff for more details.
`

func main() {
	kctx := kong.Parse(&cli, benchstatVars, benchVars,
		kong.Description(strings.TrimSpace(description)),
	)
	tmpl, err := template.New("").Parse(cli.BenchArgs)
	kctx.FatalIfErrorf(err)
	var benchArgs bytes.Buffer
	err = tmpl.Execute(&benchArgs, cli)
	kctx.FatalIfErrorf(err)

	bd := &internal.Benchdiff{
		BenchCmd:   cli.BenchCmd,
		BenchArgs:  benchArgs.String(),
		ResultsDir: cli.CacheDir,
		BaseRef:    cli.BaseRef,
		Path:       ".",
		Writer:     os.Stdout,
		Benchstat:  buildBenchstat(cli.BenchstatOpts),
		Force:      cli.ForceBase,
		GitCmd:     cli.GitCmd,
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
	})
	kctx.FatalIfErrorf(err)
	if result.HasChangeType(internal.DegradingChange) {
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

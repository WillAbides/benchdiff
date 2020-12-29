package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/alecthomas/kong"
	"github.com/willabides/benchdiff"
	pkgbenchstat "github.com/willabides/benchdiff/pkg/benchstat"
	"golang.org/x/perf/benchstat"
)

const defaultBenchArgsTmpl = `test -bench {{.Bench}} -run '^$' -benchmem -count {{.BenchCount}} {{.Packages}}`

var benchstatVars = kong.Vars{
	"AlphaDefault":     "0.05",
	"AlphaHelp":        `consider change significant if p < α (default 0.05)`,
	"CSVHelp":          `print results in CSV form`,
	"DeltaTestHelp":    `significance test to apply to delta: utest, ttest, or none`,
	"DeltaTestDefault": `utest`,
	"DeltaTestEnum":    `utest,ttest,none`,
	"GeomeanHelp":      `print the geometric mean of each file`,
	"HTMLHelp":         `print results as an HTML table`,
	"NorangeHelp":      `suppress range columns (CSV only)`,
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
	Norange     bool    `kong:"help=${NorangeHelp}"`
	ReverseSort bool    `kong:"help=${ReverseSortHelp}"`
	Sort        string  `kong:"help=${SortHelp},enum=${SortEnum},default=none"`
	Split       string  `kong:"help=${SplitHelp},default=${SplitDefault}"`
}

var benchVars = kong.Vars{
	"BenchCmdDefault":     `go`,
	"BenchArgsDefault":    defaultBenchArgsTmpl,
	"ResultsDirDefault":   filepath.FromSlash("./tmp"),
	"BenchCountHelp":      `Run each benchmark n times.`,
	"BenchHelp":           `Run only those benchmarks matching a regular expression.`,
	"BenchArgsHelp":       `Use these arguments to run benchmarks. It may be a template.`,
	"PackagesHelp":        `Run benchmarks in these packages.`,
	"BenchCmdHelp":        `The go command to use for benchmarks.`,
	"ResultsDirHelp":      `The directory where benchmark output will be deposited.`,
	"BaseRefHelp":         `The git ref to be used as a baseline.`,
	"ForceBaseHelp":       `Rerun benchmarks on the base reference even if the output already exists.`,
	"DegradationExitHelp": `Exit code when there is a degradation in the results.`,
}

var cli struct {
	BaseRef         string        `kong:"default=HEAD,help=${BaseRefHelp}"`
	Bench           string        `kong:"default='.',help=${BenchHelp}"`
	BenchArgs       string        `kong:"default=${BenchArgsDefault},help=${BenchArgsHelp}"`
	BenchCmd        string        `kong:"default=${BenchCmdDefault},help=${BenchCmdHelp}"`
	BenchCount      int           `kong:"default=10,help=${BenchCountHelp}"`
	ForceBase       bool          `kong:"help=${ForceBaseHelp}"`
	Packages        string        `kong:"default='./...',help=${PackagesHelp}"`
	ResultsDir      string        `kong:"type=dir,default=${ResultsDirDefault},help=${ResultsDirHelp}"`
	DegradationExit int           `kong:"type=on-degradation,default=0,help=${DegradationExitHelp}"`
	BenchstatOpts   benchstatOpts `kong:"embed"`
}

func main() {
	kctx := kong.Parse(&cli, benchstatVars, benchVars)
	tmpl, err := template.New("").Parse(cli.BenchArgs)
	kctx.FatalIfErrorf(err)
	var benchArgs bytes.Buffer
	err = tmpl.Execute(&benchArgs, cli)
	kctx.FatalIfErrorf(err)

	differ := &benchdiff.Differ{
		BenchCmd:   cli.BenchCmd,
		BenchArgs:  benchArgs.String(),
		ResultsDir: cli.ResultsDir,
		BaseRef:    cli.BaseRef,
		Path:       ".",
		Writer:     os.Stdout,
		Benchstat:  buildBenchstat(cli.BenchstatOpts),
		Force:      cli.ForceBase,
	}
	result, err := differ.Run()
	kctx.FatalIfErrorf(err)
	err = differ.OutputResult(result)
	kctx.FatalIfErrorf(err)
	if result.HasChangeType(benchdiff.DegradingChange) {
		os.Exit(cli.DegradationExit)
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

func buildBenchstat(opts benchstatOpts) *pkgbenchstat.Benchstat {
	order := sortOpts[opts.Sort]
	reverse := opts.ReverseSort
	if order == nil {
		reverse = false
	}
	formatter := pkgbenchstat.TextFormatter(nil)
	if opts.CSV {
		formatter = pkgbenchstat.CSVFormatter(&pkgbenchstat.CSVFormatterOptions{
			NoRange: opts.Norange,
		})
	}
	if opts.HTML {
		formatter = pkgbenchstat.HTMLFormatter(nil)
	}

	return &pkgbenchstat.Benchstat{
		DeltaTest:       deltaTestOpts[opts.DeltaTest],
		Alpha:           opts.Alpha,
		AddGeoMean:      opts.Geomean,
		SplitBy:         strings.Split(opts.Split, ","),
		Order:           order,
		ReverseOrder:    reverse,
		OutputFormatter: formatter,
	}
}

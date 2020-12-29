package benchdiff

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	pkgbenchstat "github.com/willabides/benchdiff/pkg/benchstat"
	"golang.org/x/perf/benchstat"
)

// Benchdiff runs benchstats and outputs their deltas
type Benchdiff struct {
	BenchCmd   string
	BenchArgs  string
	ResultsDir string
	BaseRef    string
	Path       string
	Writer     io.Writer
	Benchstat  *pkgbenchstat.Benchstat
	Force      bool
	JSONOutput bool
}

func (c *Benchdiff) baseOutputFile() (string, error) {
	runner := &gitRunner{
		repoPath: c.Path,
	}
	revision, err := runner.run("rev-parse", c.BaseRef)
	if err != nil {
		return "", err
	}
	revision = bytes.TrimSpace(revision)
	name := fmt.Sprintf("benchstatter-%s.out", string(revision))
	return filepath.Join(c.ResultsDir, name), nil
}

type runBenchmarksResults struct {
	worktreeOutputFile string
	baseOutputFile     string
	benchmarkCmd       string
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	if err != nil {
		return !os.IsNotExist(err)
	}
	return true
}

func (c *Benchdiff) runBenchmarks() (result *runBenchmarksResults, err error) {
	result = new(runBenchmarksResults)
	worktreeFilename := filepath.Join(c.ResultsDir, "benchstatter-worktree.out")
	worktreeFile, err := os.Create(worktreeFilename)
	if err != nil {
		return nil, err
	}
	defer func() {
		cErr := worktreeFile.Close()
		if err == nil {
			err = cErr
		}
	}()

	cmd := exec.Command(c.BenchCmd, strings.Fields(c.BenchArgs)...) //nolint:gosec // this is fine
	result.benchmarkCmd = cmd.String()
	cmd.Stdout = worktreeFile
	err = cmd.Run()
	if err != nil {
		return nil, err
	}

	baseFilename, err := c.baseOutputFile()
	if err != nil {
		return nil, err
	}

	result.baseOutputFile = baseFilename
	result.worktreeOutputFile = worktreeFilename

	if fileExists(baseFilename) && !c.Force {
		return result, nil
	}

	baseFile, err := os.Create(baseFilename)
	if err != nil {
		return nil, err
	}
	defer func() {
		cErr := baseFile.Close()
		if err == nil {
			err = cErr
		}
	}()

	baseCmd := exec.Command(c.BenchCmd, strings.Fields(c.BenchArgs)...) //nolint:gosec // this is fine
	baseCmd.Stdout = baseFile
	var baseCmdErr error
	runner := &refRunner{
		ref: c.BaseRef,
		gitRunner: gitRunner{
			repoPath:      c.Path,
			gitExecutable: "",
		},
	}
	err = runner.run(func() {
		baseCmdErr = baseCmd.Run()
	})
	if err != nil {
		return nil, err
	}

	if baseCmdErr != nil {
		return nil, err
	}

	return result, nil
}

// Run runs the Benchdiff
func (c *Benchdiff) Run() (*RunResult, error) {
	err := os.MkdirAll(c.ResultsDir, 0o700)
	if err != nil {
		return nil, err
	}
	res, err := c.runBenchmarks()
	if err != nil {
		return nil, err
	}
	collection, err := c.Benchstat.Run(res.baseOutputFile, res.worktreeOutputFile)
	if err != nil {
		return nil, err
	}
	result := &RunResult{
		benchCmd: res.benchmarkCmd,
		tables:   collection.Tables(),
	}
	return result, nil
}

// RunResult is the result of a Run
type RunResult struct {
	benchCmd string
	tables   []*benchstat.Table
}

// RunResultOutputOptions options for RunResult.WriteOutput
type RunResultOutputOptions struct {
	BenchstatFormatter pkgbenchstat.OutputFormatter // default benchstat.TextFormatter(nil)
	OutputFormat       string                       // one of json or human. default: human
}

// WriteOutput outputs the result
func (r *RunResult) WriteOutput(w io.Writer, opts *RunResultOutputOptions) error {
	if opts == nil {
		opts = new(RunResultOutputOptions)
	}
	finalOpts := &RunResultOutputOptions{
		BenchstatFormatter: pkgbenchstat.TextFormatter(nil),
		OutputFormat:       "human",
	}
	if opts.BenchstatFormatter != nil {
		finalOpts.BenchstatFormatter = opts.BenchstatFormatter
	}

	if opts.OutputFormat != "" {
		finalOpts.OutputFormat = opts.OutputFormat
	}

	var benchstatBuf bytes.Buffer
	err := finalOpts.BenchstatFormatter(&benchstatBuf, r.tables)
	if err != nil {
		return err
	}

	var fn func(io.Writer, string) error
	switch finalOpts.OutputFormat {
	case "human":
		fn = r.writeHumanResult
	case "json":
		fn = r.writeJSONResult
	default:
		return fmt.Errorf("unknown OutputFormat")
	}
	return fn(w, benchstatBuf.String())
}

func (r *RunResult) writeJSONResult(w io.Writer, benchstatResult string) error {
	type runResultJSON struct {
		BenchCommand    string `json:"bench_command,omitempty"`
		BenchstatResult string `json:"benchstat_result,omitempty"`
	}
	o := runResultJSON{
		BenchCommand:    r.benchCmd,
		BenchstatResult: benchstatResult,
	}
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(&o)
}

func (r *RunResult) writeHumanResult(w io.Writer, benchstatResult string) error {
	var err error
	if r.benchCmd != "" {
		_, err = fmt.Fprintf(w, "bench command:\n  %s\n", r.benchCmd)
		if err != nil {
			return err
		}
	}
	if benchstatResult != "" {
		_, err = fmt.Fprintf(w, "result:\n\n%s\n", benchstatResult)
		if err != nil {
			return err
		}
	}
	return nil
}

// HasChangeType returns true if the result has at least one change with the given type
func (r *RunResult) HasChangeType(changeType BenchmarkChangeType) bool {
	for _, table := range r.tables {
		for _, row := range table.Rows {
			if row.Change == int(changeType) {
				return true
			}
		}
	}
	return false
}

// BenchmarkChangeType is whether a change is an improvement or degradation
type BenchmarkChangeType int

// BenchmarkChangeType values
const (
	DegradingChange     = -1 // represents a statistically significant degradation
	InsignificantChange = 0  // represents no statistically significant change
	ImprovingChange     = 1  // represents a statistically significant improvement
)

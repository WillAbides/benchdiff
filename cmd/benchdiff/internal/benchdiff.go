package internal

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/willabides/benchdiff/pkg/benchstatter"
	"golang.org/x/crypto/sha3"
	"golang.org/x/perf/benchstat"
)

// Benchdiff runs benchstats and outputs their deltas
type Benchdiff struct {
	BenchCmd   string
	BenchArgs  string
	ResultsDir string
	BaseRef    string
	Path       string
	GitCmd     string
	Writer     io.Writer
	Benchstat  *benchstatter.Benchstat
	Force      bool
	JSONOutput bool
}

type runBenchmarksResults struct {
	worktreeOutputFile string
	baseOutputFile     string
	benchmarkCmd       string
	headSHA            string
	baseSHA            string
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	if err != nil {
		return !os.IsNotExist(err)
	}
	return true
}

func (c *Benchdiff) gitRunner() *gitRunner {
	return &gitRunner{
		gitExecutable: c.GitCmd,
		repoPath:      c.Path,
	}
}

func (c *Benchdiff) baseRefRunner() *refRunner {
	gr := c.gitRunner()
	return &refRunner{
		ref:       c.BaseRef,
		gitRunner: *gr,
	}
}

func (c *Benchdiff) cacheKey() string {
	var b []byte
	b = append(b, []byte(c.BenchCmd)...)
	b = append(b, []byte(c.BenchArgs)...)
	sum := sha3.Sum224(b)
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

func (c *Benchdiff) runBenchmarks() (result *runBenchmarksResults, err error) {
	result = new(runBenchmarksResults)
	worktreeFilename := filepath.Join(c.ResultsDir, "benchdiff-worktree.out")
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

	headSHA, err := c.gitRunner().getRefSha("HEAD")
	if err != nil {
		return nil, err
	}
	baseSHA, err := c.gitRunner().getRefSha(c.BaseRef)
	if err != nil {
		return nil, err
	}

	baseFilename := fmt.Sprintf("benchdiff-%s-%s.out", baseSHA, c.cacheKey())
	baseFilename = filepath.Join(c.ResultsDir, baseFilename)
	result.headSHA = headSHA
	result.baseSHA = baseSHA
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

	err = c.baseRefRunner().run(func() {
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
		headSHA:  res.headSHA,
		baseSHA:  res.baseSHA,
		benchCmd: res.benchmarkCmd,
		tables:   collection.Tables(),
	}
	return result, nil
}

// RunResult is the result of a Run
type RunResult struct {
	headSHA  string
	baseSHA  string
	benchCmd string
	tables   []*benchstat.Table
}

// RunResultOutputOptions options for RunResult.WriteOutput
type RunResultOutputOptions struct {
	BenchstatFormatter benchstatter.OutputFormatter // default benchstatter.TextFormatter(nil)
	OutputFormat       string                       // one of json or human. default: human
}

// WriteOutput outputs the result
func (r *RunResult) WriteOutput(w io.Writer, opts *RunResultOutputOptions) error {
	if opts == nil {
		opts = new(RunResultOutputOptions)
	}
	finalOpts := &RunResultOutputOptions{
		BenchstatFormatter: benchstatter.TextFormatter(nil),
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
		HeadSHA         string `json:"head_sha,omitempty"`
		BaseSHA         string `json:"base_sha,omitempty"`
		BenchstatOutput string `json:"benchstat_output,omitempty"`
	}
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(&runResultJSON{
		BenchCommand:    r.benchCmd,
		BenchstatOutput: benchstatResult,
		HeadSHA:         r.headSHA,
		BaseSHA:         r.baseSHA,
	})
}

func (r *RunResult) writeHumanResult(w io.Writer, benchstatResult string) error {
	var err error
	_, err = fmt.Fprintf(w, "bench command:\n  %s\n", r.benchCmd)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(w, "HEAD sha:\n  %s\n", r.headSHA)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(w, "base sha:\n  %s\n", r.baseSHA)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(w, "benchstat output:\n\n%s\n", benchstatResult)
	if err != nil {
		return err
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

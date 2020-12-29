package benchdiff

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	pkgbenchstat "github.com/willabides/benchdiff/pkg/benchstat"
	"golang.org/x/perf/benchstat"
)

// Differ runs benchstats and outputs their deltas
type Differ struct {
	BenchCmd   string
	BenchArgs  string
	ResultsDir string
	BaseRef    string
	Path       string
	Writer     io.Writer
	Benchstat  *pkgbenchstat.Benchstat
	Force      bool
}

func (c *Differ) baseOutputFile() (string, error) {
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
	worktreeOutputFile, baseOutputFile string
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	if err != nil {
		return !os.IsNotExist(err)
	}
	return true
}

func (c *Differ) runBenchmarks() (result *runBenchmarksResults, err error) {
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
	fmt.Println(c.BenchArgs)
	cmd.Stdout = worktreeFile
	err = cmd.Run()
	if err != nil {
		return nil, err
	}

	baseFilename, err := c.baseOutputFile()
	if err != nil {
		return nil, err
	}

	result = &runBenchmarksResults{
		worktreeOutputFile: worktreeFilename,
		baseOutputFile:     baseFilename,
	}

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

// Run runs the Differ
func (c *Differ) Run() (*RunResult, error) {
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
		tables: collection.Tables(),
	}
	return result, nil
}

// OutputResult outputs a Run result
func (c *Differ) OutputResult(runResult *RunResult) error {
	return c.Benchstat.OutputTables(runResult.tables)
}

// RunResult is the result of a Run
type RunResult struct {
	tables []*benchstat.Table
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

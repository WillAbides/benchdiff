package benchdiff

import (
	"bytes"
	"fmt"
	"os/exec"
	"path/filepath"
)

type gitRunner struct {
	repoPath      string
	gitExecutable string
}

func (r *gitRunner) run(args ...string) ([]byte, error) {
	executable := "git"
	if r.gitExecutable != "" {
		executable = r.gitExecutable
	}
	cmd := exec.Command(executable, args...) //nolint:gosec // this is fine
	var err error
	cmd.Dir, err = filepath.Abs(r.repoPath)
	if err != nil {
		return nil, err
	}
	return cmd.Output()
}

type refRunner struct {
	gitRunner gitRunner
	ref       string
}

func (r *refRunner) stashAndReset() (revert func() error, err error) {
	revert = func() error {
		return nil
	}
	stash, err := r.gitRunner.run("stash", "create", "--quiet")
	if err != nil {
		return nil, err
	}
	stash = bytes.TrimSpace(stash)
	if len(stash) > 0 {
		revert = func() error {
			_, revertErr := r.gitRunner.run("stash", "apply", "--quiet", string(stash))
			return revertErr
		}
	}
	_, err = r.gitRunner.run("reset", "--hard", "--quiet")
	if err != nil {
		return nil, err
	}
	return revert, nil
}

func (r *refRunner) run(fn func()) error {
	origRef, err := r.gitRunner.run("rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return err
	}
	origRef = bytes.TrimSpace(origRef)
	unstash, err := r.stashAndReset()
	if err != nil {
		return err
	}
	defer func() {
		unstashErr := unstash()
		if unstashErr != nil {
			panic(unstashErr)
		}
	}()
	_, err = r.gitRunner.run("checkout", "--quiet", r.ref)
	if err != nil {
		return err
	}
	defer func() {
		_, cerr := r.gitRunner.run("checkout", "--quiet", string(origRef))
		if cerr != nil {
			if exitErr, ok := cerr.(*exec.ExitError); ok {
				fmt.Println(string(exitErr.Stderr))
			}
			fmt.Println(cerr)
		}
	}()
	fn()
	return nil
}

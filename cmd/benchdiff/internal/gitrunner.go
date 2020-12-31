package internal

import (
	"bytes"
	"fmt"
	"os/exec"
	"path/filepath"
	"time"
)

func runGitCmd(gitCmd, repoPath string, args ...string) ([]byte, error) {
	cmd := exec.Command(gitCmd, args...) //nolint:gosec // this is fine
	var err error
	cmd.Dir, err = filepath.Abs(repoPath)
	if err != nil {
		return nil, err
	}

	b, err := cmd.Output()
	if exitErr, ok := err.(*exec.ExitError); ok {
		err = fmt.Errorf("error running git command: %s", string(exitErr.Stderr))
	}
	b = bytes.TrimSpace(b)
	return b, err
}

func stashAndReset(gitCmd, repoPath string) (revert func() error, err error) {
	revert = func() error {
		return nil
	}
	stash, err := runGitCmd(gitCmd, repoPath, "stash", "create", "--quiet")
	if err != nil {
		return nil, err
	}
	stash = bytes.TrimSpace(stash)
	if len(stash) > 0 {
		revert = func() error {
			_, revertErr := runGitCmd(gitCmd, repoPath, "stash", "apply", "--quiet", string(stash))
			return revertErr
		}
	}
	_, err = runGitCmd(gitCmd, repoPath, "reset", "--hard", "--quiet")
	if err != nil {
		return nil, err
	}
	return revert, nil
}

func runAtGitRef(gitCmd, repoPath, ref string, pause time.Duration, fn func()) error {
	origRef, err := runGitCmd(gitCmd, repoPath, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return err
	}
	origRef = bytes.TrimSpace(origRef)
	unstash, err := stashAndReset(gitCmd, repoPath)
	if err != nil {
		return err
	}
	defer func() {
		unstashErr := unstash()
		if unstashErr != nil {
			panic(unstashErr)
		}
	}()
	_, err = runGitCmd(gitCmd, repoPath, "checkout", "--quiet", ref)
	if err != nil {
		return err
	}
	defer func() {
		_, cerr := runGitCmd(gitCmd, repoPath, "checkout", "--quiet", string(origRef))
		if cerr != nil {
			if exitErr, ok := cerr.(*exec.ExitError); ok {
				fmt.Println(string(exitErr.Stderr))
			}
			fmt.Println(cerr)
		}
	}()
	time.Sleep(pause)
	fn()
	return nil
}

package internal

import (
	"bytes"
	"fmt"
	"log"
	"os/exec"
	"time"
)

func runGitCmd(debug *log.Logger, gitCmd, repoPath string, args ...string) ([]byte, error) {
	var stdout bytes.Buffer
	cmd := exec.Command(gitCmd, args...) //nolint:gosec // this is fine
	cmd.Stdout = &stdout
	cmd.Dir = repoPath
	err := runCmd(cmd, debug)
	return bytes.TrimSpace(stdout.Bytes()), err
}

func stashAndReset(debug *log.Logger, gitCmd, repoPath string) (revert func() error, err error) {
	revert = func() error {
		return nil
	}
	stash, err := runGitCmd(debug, gitCmd, repoPath, "stash", "create")
	if err != nil {
		return nil, err
	}
	stash = bytes.TrimSpace(stash)
	if len(stash) > 0 {
		revert = func() error {
			_, revertErr := runGitCmd(debug, gitCmd, repoPath, "stash", "apply", string(stash))
			return revertErr
		}
	}
	_, err = runGitCmd(debug, gitCmd, repoPath, "reset", "--hard")
	if err != nil {
		return nil, err
	}
	return revert, nil
}

func runAtGitRef(debug *log.Logger, gitCmd, repoPath, ref string, pause time.Duration, fn func()) error {
	origRef, err := runGitCmd(nil, gitCmd, repoPath, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return err
	}
	origRef = bytes.TrimSpace(origRef)
	unstash, err := stashAndReset(debug, gitCmd, repoPath)
	if err != nil {
		return err
	}
	defer func() {
		unstashErr := unstash()
		if unstashErr != nil {
			panic(unstashErr)
		}
	}()
	_, err = runGitCmd(debug, gitCmd, repoPath, "checkout", "--quiet", ref)
	if err != nil {
		return err
	}
	defer func() {
		_, cerr := runGitCmd(debug, gitCmd, repoPath, "checkout", "--quiet", string(origRef))
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

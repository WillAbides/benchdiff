package internal

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os/exec"
	"path/filepath"
	"time"
)

func runGitCmd(debug *log.Logger, gitCmd, repoPath string, args ...string) ([]byte, error) {
	if debug == nil {
		debug = log.New(ioutil.Discard, "", 0)
	}

	cmd := exec.Command(gitCmd, args...) //nolint:gosec // this is fine
	var err error
	cmd.Dir, err = filepath.Abs(repoPath)
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	out := io.MultiWriter(&buf, debug.Writer())
	debug.Printf(cmd.String())
	cmd.Stdout = out
	err = cmd.Run()
	if exitErr, ok := err.(*exec.ExitError); ok {
		err = fmt.Errorf("error running git command: %s", string(exitErr.Stderr))
	}
	return bytes.TrimSpace(buf.Bytes()), err
}

func stashAndReset(debug *log.Logger, gitCmd, repoPath string) (revert func() error, err error) {
	revert = func() error {
		return nil
	}
	stash, err := runGitCmd(debug, gitCmd, repoPath, "stash", "create", "--quiet")
	if err != nil {
		return nil, err
	}
	stash = bytes.TrimSpace(stash)
	if len(stash) > 0 {
		revert = func() error {
			_, revertErr := runGitCmd(debug, gitCmd, repoPath, "stash", "apply", "--quiet", string(stash))
			return revertErr
		}
	}
	_, err = runGitCmd(debug, gitCmd, repoPath, "reset", "--hard", "--quiet")
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

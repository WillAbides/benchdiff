package internal

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

var preserveTmpDir bool

func tmpDir(t *testing.T) string {
	t.Helper()
	projectTmp := filepath.FromSlash("../../../tmp")

	err := os.MkdirAll(projectTmp, 0o700)
	assert.NoError(t, err)
	tmpdir, err := ioutil.TempDir(projectTmp, "")
	assert.NoError(t, err)
	t.Cleanup(func() {
		if preserveTmpDir {
			t.Logf("tmp dir preserved at %s", tmpdir)
			return
		}
		assert.NoError(t, os.RemoveAll(tmpdir))
	})
	return tmpdir
}

func mustSetEnv(t *testing.T, env map[string]string) {
	t.Helper()
	for k, v := range env {
		assert.NoError(t, os.Setenv(k, v))
	}
}

func mustGit(t *testing.T, repoPath string, args ...string) []byte {
	t.Helper()
	mustSetEnv(t, map[string]string{
		"GIT_AUTHOR_NAME":     "author",
		"GIT_AUTHOR_EMAIL":    "author@localhost",
		"GIT_COMMITTER_NAME":  "committer",
		"GIT_COMMITTER_EMAIL": "committer@localhost",
	})
	got, err := runGitCmd(nil, "git", repoPath, args...)
	assert.NoErrorf(t, err, "error running git:\noutput: %v", string(got))
	return got
}

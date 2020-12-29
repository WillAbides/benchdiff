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

func mustSetEnv(t *testing.T, key, value string) {
	t.Helper()
	assert.NoError(t, os.Setenv(key, value))
}

func mustGit(t *testing.T, repoPath string, args ...string) []byte {
	t.Helper()
	mustSetEnv(t, "GIT_AUTHOR_NAME", "author")
	mustSetEnv(t, "GIT_AUTHOR_EMAIL", "author@localhost")
	mustSetEnv(t, "GIT_COMMITTER_NAME", "committer")
	mustSetEnv(t, "GIT_COMMITTER_EMAIL", "committer@localhost")
	runner := &gitRunner{
		repoPath: repoPath,
	}
	got, err := runner.run(args...)
	assert.NoErrorf(t, err, "error running git:\noutput: %v", string(got))
	return got
}

package benchdiff

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

var doNotDeleteTmpDir bool

func tmpDir(t *testing.T) string {
	t.Helper()
	projectTmp := filepath.FromSlash("./tmp")

	err := os.MkdirAll(projectTmp, 0o700)
	require.NoError(t, err)
	tmpdir, err := ioutil.TempDir(projectTmp, "")
	require.NoError(t, err)
	t.Cleanup(func() {
		if !doNotDeleteTmpDir {
			require.NoError(t, os.RemoveAll(tmpdir))
		}
	})
	return tmpdir
}

func mustGit(t *testing.T, repoPath string, args ...string) []byte {
	t.Helper()
	runner := &gitRunner{
		repoPath: repoPath,
	}
	got, err := runner.run(args...)
	require.NoErrorf(t, err, "error running git:\noutput: %v", string(got))
	return got
}

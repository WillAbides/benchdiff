package internal

import (
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_runAtGitRef(t *testing.T) {
	dir := t.TempDir()
	fooPath := filepath.Join(dir, "foo")
	err := ioutil.WriteFile(fooPath, []byte("OG content"), 0o600)
	require.NoError(t, err)
	mustGit(t, dir, "init")
	mustGit(t, dir, "add", "foo")
	mustGit(t, dir, "commit", "-m", "ignore me")
	untrackedPath := filepath.Join(dir, "untracked")
	err = ioutil.WriteFile(untrackedPath, []byte("untracked"), 0o600)
	require.NoError(t, err)
	err = ioutil.WriteFile(fooPath, []byte("new content"), 0o600)
	require.NoError(t, err)
	fn := func(workDir string) {
		var got []byte
		untrackedPath := filepath.Join(workDir, "untracked")
		_, err = ioutil.ReadFile(untrackedPath)
		require.Error(t, err)
		wdFooPath := filepath.Join(workDir, "foo")
		got, err = ioutil.ReadFile(wdFooPath)
		require.NoError(t, err)
		require.Equal(t, "OG content", string(got))
	}
	err = runAtGitRef(nil, "git", dir, "HEAD", fn)
	require.NoError(t, err)
	got, err := ioutil.ReadFile(fooPath)
	require.NoError(t, err)
	require.Equal(t, "new content", string(got))
}

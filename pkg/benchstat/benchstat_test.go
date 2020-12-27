package benchstat

import (
	"bytes"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBenchstat_Run(t *testing.T) {
	worktreeFile := filepath.FromSlash("./testdata/outputs/benchstatter-worktree.out")
	baseFile := filepath.FromSlash("./testdata/outputs/benchstatter-1.out")
	var buf bytes.Buffer
	bs := &Benchstat{
		Writer: &buf,
	}
	err := bs.Run(worktreeFile, baseFile)
	require.NoError(t, err)

	want := `name         old time/op    new time/op    delta
DoNothing-8    1.31ms ±13%   10.89ms ± 7%  +728.87%  (p=0.000 n=10+10)

name         old alloc/op   new alloc/op   delta
DoNothing-8     32.2B ± 2%     11.4B ± 5%   -64.48%  (p=0.000 n=9+9)

name         old allocs/op  new allocs/op  delta
DoNothing-8      0.00           0.00           ~     (all equal)
`
	require.Equal(t, want, buf.String())
}

package benchstatter

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_formatGroup(t *testing.T) {
	input := "pkg:encoding/gob goos:darwin note:hw acceleration enabled foo:bar"
	want := `pkg: encoding/gob
goos: darwin
note: hw acceleration enabled
foo: bar`
	got := formatGroup(input)
	require.Equal(t, want, got)
}

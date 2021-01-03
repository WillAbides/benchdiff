package helpprinter

import (
	"bytes"
	"strings"
	"testing"

	"github.com/alecthomas/kong"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func flagsFromGroupTags(tags []string) []*kong.Flag {
	flags := make([]*kong.Flag, len(tags))
	for i, tag := range tags {
		flags[i] = &kong.Flag{
			Group: tag,
			Value: &kong.Value{
				Position: i,
			},
		}
	}
	return flags
}

func assertFlagOrder(t *testing.T, wantOrder []int, flags []*kong.Flag) bool {
	t.Helper()
	got := make([]int, len(flags))
	for i, flag := range flags {
		got[i] = flag.Value.Position
	}
	if wantOrder == nil {
		wantOrder = []int{}
	}
	return assert.Equal(t, wantOrder, got)
}

func Test_groupFlagsByTag(t *testing.T) {
	for _, td := range []struct {
		name          string
		flagGroupTags []string
		want          [][]int
	}{
		{
			name: "empty",
			want: [][]int{{}},
		},
		{
			name:          "no tags",
			flagGroupTags: []string{"", "", ""},
			want:          [][]int{{0, 1, 2}},
		},
		{
			name:          "empty goes first",
			flagGroupTags: []string{"a", "", "a"},
			want:          [][]int{{1}, {0, 2}},
		},
		{
			name:          "multiple groups",
			flagGroupTags: []string{"a", "", "a", "b", "b", "a"},
			want:          [][]int{{1}, {0, 2, 5}, {3, 4}},
		},
		{
			name:          "no empty",
			flagGroupTags: []string{"a", "a", "b", "b", "a"},
			want:          [][]int{{0, 1, 4}, {2, 3}},
		},
		{
			name:          "space isn't empty",
			flagGroupTags: []string{"a", "a", "b", " ", "a"},
			want:          [][]int{{0, 1, 4}, {2}, {3}},
		},
	} {
		t.Run(td.name, func(t *testing.T) {
			flags := flagsFromGroupTags(td.flagGroupTags)
			got := groupFlagsByTag(flags)
			assert.Len(t, got, len(td.want))
			for i := range td.want {
				assertFlagOrder(t, td.want[i], got[i])
			}
		})
	}
}

func Test_sortFlagsByGroup(t *testing.T) {
	for _, td := range []struct {
		name          string
		flagGroupTags []string
		wantOrder     []int
	}{
		{
			name: "empty",
		},
		{
			name:          "no tags",
			flagGroupTags: []string{"", "", ""},
			wantOrder:     []int{0, 1, 2},
		},
		{
			name:          "empty goes first",
			flagGroupTags: []string{"a", "", "a"},
			wantOrder:     []int{1, 0, 2},
		},
		{
			name:          "multiple groups",
			flagGroupTags: []string{"a", "", "a", "b", "b", "a"},
			wantOrder:     []int{1, 0, 2, 5, 3, 4},
		},
		{
			name:          "no empty",
			flagGroupTags: []string{"a", "a", "b", "b", "a"},
			wantOrder:     []int{0, 1, 4, 2, 3},
		},
		{
			name:          "space isn't empty",
			flagGroupTags: []string{"a", "a", "b", " ", "a"},
			wantOrder:     []int{0, 1, 4, 2, 3},
		},
	} {
		t.Run(td.name, func(t *testing.T) {
			flags := flagsFromGroupTags(td.flagGroupTags)
			sortFlagsByGroup(flags)
			assertFlagOrder(t, td.wantOrder, flags)
		})
	}
}

func TestNewHelpPrinter(t *testing.T) {
	var cli struct {
		A string  `kong:"help='this is a'"`
		B string  `kong:"group=x,help='this is b'"`
		C string  `kong:"group=y,help='this is c'"`
		D []int64 `kong:"group=x,help='this is d'"`
		E string  `kong:"group=z,help='this is e',short=E"`
	}

	var buf bytes.Buffer
	k, err := kong.New(&cli,
		kong.Writers(&buf, nil),
		kong.Name("appname"),
		kong.Vars{
			"xGroupHelp": `group x is like this`,
			"zGroupHelp": `group z is like this`,
		},
	)
	require.NoError(t, err)
	kctx, err := kong.Trace(k, nil)
	require.NoError(t, err)

	printer := NewHelpPrinter(nil)
	err = printer(kong.HelpOptions{}, kctx)
	require.NoError(t, err)

	want := `
Usage: appname

Flags:
  -h, --help        Show context-sensitive help.
      --a=STRING    this is a

  group x is like this
      --b=STRING    this is b
      --d=D,...     this is d

      --c=STRING    this is c

  group z is like this
  -E, --e=STRING    this is e
`

	require.Equal(t, strings.TrimSpace(want), strings.TrimSpace(buf.String()))
}

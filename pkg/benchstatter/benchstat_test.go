package benchstatter

import (
	"bytes"
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/perf/benchstat"
)

func TestBenchstat_Run(t *testing.T) {
	pwd, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir("testdata"))
	t.Cleanup(func() {
		t.Helper()
		require.NoError(t, os.Chdir(pwd))
	})
	for _, td := range []struct {
		golden string
		base   string
		head   string
		b      *Benchstat
	}{
		{
			golden: "example",
			base:   "exampleold.txt",
			head:   "examplenew.txt",
			b:      new(Benchstat),
		},
		{
			golden: "examplehtml",
			base:   "exampleold.txt",
			head:   "examplenew.txt",
			b: &Benchstat{
				OutputFormatter: HTMLFormatter(nil),
			},
		},
		{
			golden: "examplecsv",
			base:   "exampleold.txt",
			head:   "examplenew.txt",
			b: &Benchstat{
				OutputFormatter: CSVFormatter(nil),
			},
		},
		{
			golden: "examplecsv-norange",
			base:   "exampleold.txt",
			head:   "examplenew.txt",
			b: &Benchstat{
				OutputFormatter: CSVFormatter(&CSVFormatterOptions{
					NoRange: true,
				}),
			},
		},
		{
			golden: "oldnew",
			base:   "old.txt",
			head:   "new.txt",
			b:      new(Benchstat),
		},
		{
			golden: "oldnewgeo",
			base:   "old.txt",
			head:   "new.txt",
			b: &Benchstat{
				AddGeoMean: true,
			},
		},
		{
			golden: "new4",
			base:   "new.txt",
			head:   "slashslash4.txt",
			b:      new(Benchstat),
		},
		{
			golden: "oldnewhtml",
			base:   "old.txt",
			head:   "new.txt",
			b: &Benchstat{
				OutputFormatter: HTMLFormatter(nil),
			},
		},
		{
			golden: "oldnewcsv",
			base:   "old.txt",
			head:   "new.txt",
			b: &Benchstat{
				OutputFormatter: CSVFormatter(nil),
			},
		},
		{
			golden: "oldnewttest",
			base:   "old.txt",
			head:   "new.txt",
			b: &Benchstat{
				DeltaTest: benchstat.TTest,
			},
		},
		{
			golden: "packages",
			base:   "packagesold.txt",
			head:   "packagesnew.txt",
			b: &Benchstat{
				SplitBy: []string{"pkg", "goos", "goarch"},
			},
		},
		{
			golden: "units",
			base:   "units-old.txt",
			head:   "units-new.txt",
			b:      new(Benchstat),
		},
		{
			golden: "zero",
			base:   "zero-old.txt",
			head:   "zero-new.txt",
			b: &Benchstat{
				DeltaTest: benchstat.NoDeltaTest,
			},
		},
		{
			golden: "namesort",
			base:   "old.txt",
			head:   "new.txt",
			b: &Benchstat{
				Order: benchstat.ByName,
			},
		},
		{
			golden: "deltasort",
			base:   "old.txt",
			head:   "new.txt",
			b: &Benchstat{
				Order: benchstat.ByDelta,
			},
		},
		{
			golden: "rdeltasort",
			base:   "old.txt",
			head:   "new.txt",
			b: &Benchstat{
				Order:        benchstat.ByDelta,
				ReverseOrder: true,
			},
		},
	} {
		t.Run(td.golden, func(t *testing.T) {
			result, err := td.b.Run(td.base, td.head)
			require.NoError(t, err)
			var buf bytes.Buffer
			err = td.b.OutputTables(&buf, result.Tables())
			require.NoError(t, err)
			want, err := ioutil.ReadFile(td.golden + ".golden")
			require.NoError(t, err)
			require.Equal(t, string(want), buf.String())
		})
	}
}

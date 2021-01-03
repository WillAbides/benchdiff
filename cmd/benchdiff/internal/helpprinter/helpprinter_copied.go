package helpprinter

import (
	"bytes"
	"fmt"
	"go/doc"
	"io"
	"strings"

	. "github.com/alecthomas/kong" //nolint:golint // dot import is so I can copy code verbatim
)

// Everything below here is copied verbatim from
// https://github.com/alecthomas/kong/blob/d78d607800e2d9a4eb100a7b48021f17196babcb/help.go
//
// The only modifications are from gofumpt
//
// Unneeded code and modified functions are moved to helpprinter.go

const (
	defaultIndent        = 2
	defaultColumnPadding = 4
)

func printApp(w *helpWriter, app *Application) {
	if !w.NoAppSummary {
		w.Printf("Usage: %s%s", app.Name, app.Summary())
	}
	printNodeDetail(w, app.Node, true)
	cmds := app.Leaves(true)
	if len(cmds) > 0 && app.HelpFlag != nil {
		w.Print("")
		if w.Summary {
			w.Printf(`Run "%s --help" for more information.`, app.Name)
		} else {
			w.Printf(`Run "%s <command> --help" for more information on a command.`, app.Name)
		}
	}
}

func printCommand(w *helpWriter, app *Application, cmd *Command) {
	if !w.NoAppSummary {
		w.Printf("Usage: %s %s", app.Name, cmd.Summary())
	}
	printNodeDetail(w, cmd, true)
	if w.Summary && app.HelpFlag != nil {
		w.Print("")
		w.Printf(`Run "%s --help" for more information.`, cmd.FullPath())
	}
}

func writeCommandList(cmds []*Node, iw *helpWriter) {
	for i, cmd := range cmds {
		if cmd.Hidden {
			continue
		}
		printCommandSummary(iw, cmd)
		if i != len(cmds)-1 {
			iw.Print("")
		}
	}
}

func writeCompactCommandList(cmds []*Node, iw *helpWriter) {
	rows := [][2]string{}
	for _, cmd := range cmds {
		if cmd.Hidden {
			continue
		}
		rows = append(rows, [2]string{cmd.Path(), cmd.Help})
	}
	writeTwoColumns(iw, rows)
}

func writeCommandTree(w *helpWriter, node *Node) {
	iw := w.Indent()
	rows := make([][2]string, 0, len(node.Children)*2)
	for i, cmd := range node.Children {
		if cmd.Hidden {
			continue
		}
		rows = append(rows, w.CommandTree(cmd, "")...)
		if i != len(node.Children)-1 {
			rows = append(rows, [2]string{"", ""})
		}
	}
	writeTwoColumns(iw, rows)
}

func printCommandSummary(w *helpWriter, cmd *Command) {
	w.Print(cmd.Summary())
	if cmd.Help != "" {
		w.Indent().Wrap(cmd.Help)
	}
}

type helpWriter struct {
	indent        string
	width         int
	lines         *[]string
	helpFormatter HelpValueFormatter
	HelpOptions
}

func (h *helpWriter) Printf(format string, args ...interface{}) {
	h.Print(fmt.Sprintf(format, args...))
}

func (h *helpWriter) Print(text string) {
	*h.lines = append(*h.lines, strings.TrimRight(h.indent+text, " "))
}

func (h *helpWriter) Indent() *helpWriter {
	return &helpWriter{indent: h.indent + "  ", lines: h.lines, width: h.width - 2, HelpOptions: h.HelpOptions, helpFormatter: h.helpFormatter}
}

func (h *helpWriter) String() string {
	return strings.Join(*h.lines, "\n")
}

func (h *helpWriter) Write(w io.Writer) error {
	for _, line := range *h.lines {
		_, err := io.WriteString(w, line+"\n")
		if err != nil {
			return err
		}
	}
	return nil
}

func (h *helpWriter) Wrap(text string) {
	w := bytes.NewBuffer(nil)
	doc.ToText(w, strings.TrimSpace(text), "", "    ", h.width)
	for _, line := range strings.Split(strings.TrimSpace(w.String()), "\n") {
		h.Print(line)
	}
}

func writePositionals(w *helpWriter, args []*Positional) {
	rows := [][2]string{}
	for _, arg := range args {
		rows = append(rows, [2]string{arg.Summary(), w.helpFormatter(arg)})
	}
	writeTwoColumns(w, rows)
}

func writeTwoColumns(w *helpWriter, rows [][2]string) {
	maxLeft := 375 * w.width / 1000
	if maxLeft < 30 {
		maxLeft = 30
	}
	// Find size of first column.
	leftSize := 0
	for _, row := range rows {
		if c := len(row[0]); c > leftSize && c < maxLeft {
			leftSize = c
		}
	}

	offsetStr := strings.Repeat(" ", leftSize+defaultColumnPadding)

	for _, row := range rows {
		buf := bytes.NewBuffer(nil)
		doc.ToText(buf, row[1], "", strings.Repeat(" ", defaultIndent), w.width-leftSize-defaultColumnPadding)
		lines := strings.Split(strings.TrimRight(buf.String(), "\n"), "\n")

		line := fmt.Sprintf("%-*s", leftSize, row[0])
		if len(row[0]) < maxLeft {
			line += fmt.Sprintf("%*s%s", defaultColumnPadding, "", lines[0])
			lines = lines[1:]
		}
		w.Print(line)
		for _, line := range lines {
			w.Printf("%s%s", offsetStr, line)
		}
	}
}

// haveShort will be true if there are short flags present at all in the help. Useful for column alignment.
func formatFlag(haveShort bool, flag *Flag) string {
	flagString := ""
	name := flag.Name
	isBool := flag.IsBool()
	if flag.Short != 0 {
		flagString += fmt.Sprintf("-%c, --%s", flag.Short, name)
	} else {
		if haveShort {
			flagString += fmt.Sprintf("    --%s", name)
		} else {
			flagString += fmt.Sprintf("--%s", name)
		}
	}
	if !isBool {
		flagString += fmt.Sprintf("=%s", flag.FormatPlaceHolder())
	}
	return flagString
}

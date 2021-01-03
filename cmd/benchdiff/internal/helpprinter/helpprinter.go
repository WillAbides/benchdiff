package helpprinter

import (
	"sort"

	"github.com/alecthomas/kong"
)

// NewHelpPrinter returns a new kong.HelpPrinter
func NewHelpPrinter(helpValueFormatter kong.HelpValueFormatter) kong.HelpPrinter {
	return func(options kong.HelpOptions, ctx *kong.Context) error {
		if ctx.Empty() {
			options.Summary = false
		}
		w := newHelpWriter(ctx, helpValueFormatter, options)
		selected := ctx.Selected()
		if selected == nil {
			printApp(w, ctx.Model)
		} else {
			printCommand(w, ctx.Model, selected)
		}
		return w.Write(ctx.Stdout)
	}
}

func sortFlagsByGroup(flags []*kong.Flag) {
	groupOrder := map[string]int{
		"": -1,
	}
	for i, flag := range flags {
		_, ok := groupOrder[flag.Group]
		if !ok {
			groupOrder[flag.Group] = i
		}
	}
	sort.SliceStable(flags, func(i, j int) bool {
		return groupOrder[flags[i].Group] < groupOrder[flags[j].Group]
	})
}

func groupFlagsByTag(flags []*kong.Flag) [][]*kong.Flag {
	// make a copy so this doesn't have the side effect of sorting flags
	fl := make([]*kong.Flag, len(flags))
	copy(fl, flags)
	sortFlagsByGroup(fl)

	groups := [][]*kong.Flag{{}}

	for i, flag := range fl {
		if i > 0 && fl[i-1].Group != flag.Group {
			groups = append(groups, []*kong.Flag{})
		}
		groups[len(groups)-1] = append(groups[len(groups)-1], flag)
	}

	return groups
}

// regroupFlags sorts each flag group by group tag then splits the group by group tag
func regroupFlags(flagGroups [][]*kong.Flag) [][]*kong.Flag {
	result := make([][]*kong.Flag, 0, len(flagGroups))
	for _, group := range flagGroups {
		result = append(result, groupFlagsByTag(group)...)
	}
	return result
}

// Below here is modified code from https://github.com/alecthomas/kong/blob/d78d607800e2d9a4eb100a7b48021f17196babcb/help.go

func newHelpWriter(ctx *kong.Context, helpFormatter kong.HelpValueFormatter, options kong.HelpOptions) *helpWriter {
	if helpFormatter == nil {
		helpFormatter = kong.DefaultHelpValueFormatter
	}
	lines := []string{}
	w := &helpWriter{
		indent:        "",
		width:         guessWidth(ctx.Stdout),
		lines:         &lines,
		helpFormatter: helpFormatter,
		HelpOptions:   options,
	}
	return w
}

func printNodeDetail(w *helpWriter, node *kong.Node, hide bool) {
	if node.Help != "" {
		w.Print("")
		w.Wrap(node.Help)
	}
	if w.Summary {
		return
	}
	if node.Detail != "" {
		w.Print("")
		w.Wrap(node.Detail)
	}
	if len(node.Positional) > 0 {
		w.Print("")
		w.Print("Arguments:")
		writePositionals(w.Indent(), node.Positional)
	}
	if flags := node.AllFlags(true); len(flags) > 0 {
		flags = regroupFlags(flags)
		w.Print("")
		w.Print("Flags:")
		writeFlags(w.Indent(), node.Vars(), flags)
	}

	cmds := node.Leaves(hide)
	if len(cmds) > 0 {
		w.Print("")
		w.Print("Commands:")
		if w.Tree {
			writeCommandTree(w, node)
		} else {
			iw := w.Indent()
			if w.Compact {
				writeCompactCommandList(cmds, iw)
			} else {
				writeCommandList(cmds, iw)
			}
		}
	}
}

func writeFlags(w *helpWriter, vars kong.Vars, groups [][]*kong.Flag) {
	haveShort := false
	for _, group := range groups {
		for _, flag := range group {
			if flag.Short != 0 {
				haveShort = true
				break
			}
		}
	}
	for i, group := range groups {
		rows := [][2]string{}
		groupHelp := ""
		if len(group) > 0 && group[0].Group != "" {
			groupHelp = vars[group[0].Group+"GroupHelp"]
		}
		if i > 0 || groupHelp != "" {
			w.Print("")
		}
		if groupHelp != "" {
			w.Print(groupHelp)
		}

		for _, flag := range group {
			if !flag.Hidden {
				rows = append(rows, [2]string{formatFlag(haveShort, flag), w.helpFormatter(flag.Value)})
			}
		}
		writeTwoColumns(w, rows)
	}
}

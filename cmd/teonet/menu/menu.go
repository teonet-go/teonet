// Copyright 2021-2023 Kirill Scherba <kirill@scherba.ru>. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Teonet CLI application menu.
package menu

import (
	"flag"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"

	"github.com/chzyer/readline"
	"github.com/teonet-go/teonet"
)

// Menu is Teonet CLI application menu
type Menu struct {
	r        *readline.Instance // Readline instance
	items    []Item             // Menu items
	appShort string             // Application short name
}
type Item interface {
	Name() string           // Get command name
	Help() string           // Get command quick help string
	Exec(line string) error // Execute command
	Compliter() []Compliter // Command compliter
}
type Compliter readline.PrefixCompleterInterface // Readline compliter type

// New creates new Teonet CLI application menu
func New(appShort string) (cmd *Menu, err error) {
	cmd = &Menu{appShort: appShort}
	return
}

// Add command (menu item)
func (m *Menu) Add(items ...Item) {
	m.items = append(m.items, items...)
}

// MakeCommand create command object for Add command. The first argument 'command' may by
// static string or dynamic function which return Items name
func (m Menu) MakeCommand(command string, help string,
	exec func(line string) error, comp ...func() []Compliter) Item {
	return simpleCommand{command, help, exec, comp}
}

// Run menu
func (m *Menu) Run() (err error) {
	m.addSystemCommands()
	m.r, err = m.newReadline()
	if err != nil {
		return
	}
	defer m.r.Close()
	space := regexp.MustCompile(`\s+`)
	for {
		var line string

		// Read line by readline
		if line, err = m.r.Readline(); err == readline.ErrInterrupt {
			if len(line) == 0 {
				break
			} else {
				continue
			}
		} else if err == io.EOF {
			break
		}

		// Trim and remove double spaces
		line = strings.TrimSpace(line)
		line = space.ReplaceAllString(line, " ")

		// Process line
		switch line {
		case "":
			continue
		case "exit", "quit":
			return
		default:
			if err = m.ExecuteCommand(line); err != nil {
				fmt.Println("error:", err)
			}
		}
	}
	return
}

// ExecuteCommand executes command using input command line
func (m Menu) ExecuteCommand(line string) (err error) {

	c := m.findCommand(line)
	if c == nil {
		err = fmt.Errorf(teonet.FmtMsgCommandNotCount, line)
		return
	}
	line = strings.TrimSpace(line[len(c.Name()):])
	return c.Exec(line)
}

// SplitSpace split line by space helper function
func (m Menu) SplitSpace(line string) (res []string) { return m.Split(line, " ") }

// SplitComma split line by comma helper function
func (m Menu) SplitComma(line string) (res []string) { return m.Split(line, ",") }

// Split line by delemiter helper function
func (m Menu) Split(line, delimiter string) (res []string) {
	if line != "" {
		res = strings.Split(line, delimiter)
	}
	return
}

// MakeCompliterFromString returns compliter created from input string slice
func (m *Menu) MakeCompliterFromString(strings []string) (cmpl []Compliter) {
	for _, s := range strings {
		cmpl = append(cmpl, readline.PcItem(s))
	}
	return cmpl
}

func (m *Menu) newReadline() (l *readline.Instance, err error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		dir = os.TempDir()
	}
	l, err = readline.NewEx(&readline.Config{
		Prompt:              "\033[31mteo»\033[0m ",
		HistoryFile:         dir + "/" + teonet.ConfigDir + "/" + m.appShort + "/readline.tmp",
		AutoComplete:        m.makeCompliter(),
		InterruptPrompt:     "^C",
		EOFPrompt:           "exit",
		HistorySearchFold:   true,
		FuncFilterInputRune: m.filterInput,
	})
	if err != nil {
		panic(err)
	}
	return
}

func (m *Menu) filterInput(r rune) (rune, bool) {
	switch r {
	// block CtrlZ feature
	case readline.CharCtrlZ:
		return r, false
	}
	return r, true
}

func (m Menu) findCommand(line string) (cmd Item) {
	lenLine := len(line)
	for _, c := range m.items {
		name := c.Name()
		lenName := len(name)
		if lenLine < lenName {
			continue
		}
		if lenLine > lenName && line[lenName] != ' ' {
			continue
		}
		cmd := line[:lenName]
		if name == cmd {
			return c
		}
	}
	return nil
}

func (m *Menu) makeCompliter() *readline.PrefixCompleter {
	var comp []readline.PrefixCompleterInterface
	for _, c := range m.items {
		compConvert := func() (comp []readline.PrefixCompleterInterface) {
			if c.Compliter() == nil {
				return
			}
			for _, cc := range c.Compliter() {
				comp = append(comp, cc)
			}
			return
		}
		compliter := readline.PcItem(c.Name(), compConvert()...)
		comp = append(comp, compliter)
	}
	return readline.NewPrefixCompleter(comp...)
}

// addSystemCommands add internel menu commands like help, etc.
func (m *Menu) addSystemCommands() { m.Add(CmdHelp{m}) }

// CmdHelp help command name
const cmdHelp = "help"

// CmdHelp help command
type CmdHelp struct{ menu *Menu }

func (c CmdHelp) Name() string { return cmdHelp }
func (c CmdHelp) Help() string { return "this help" }
func (c CmdHelp) Exec(line string) (err error) {
	flags := flag.NewFlagSet(c.Name(), flag.ContinueOnError)
	flags.Parse(c.menu.SplitSpace(line))
	args := flags.Args()
	if len(args) == 1 {
		c.menu.ExecuteCommand(args[0] + " " + cmdHelp)
		return
	}

	fmt.Print("Usage: teocli <command> [arguments]\n\nThe commands are:\n\n")
	lenMax := 0
	for _, c := range c.menu.items {
		lenName := len(c.Name())
		if lenName > lenMax {
			lenMax = lenName
		}
	}
	for _, c := range c.menu.items {
		fmt.Printf("%-*s \t%s\n", lenMax, c.Name(), c.Help())
	}
	fmt.Print("\nUse \"help <command>\" for more information about that command.\n\n")
	return
}
func (c CmdHelp) Compliter() (cmpl []Compliter) {
	for _, item := range c.menu.items {
		n := item.Name()
		if n == cmdHelp {
			continue
		}
		cmpl = append(cmpl, readline.PcItem(n))
	}
	return
}

// simpleCommand used in MakeItem
type simpleCommand struct {
	name       string
	help       string
	exec       func(line string) error
	compliters []func() []Compliter
}

// Command return command (menu item) name
func (s simpleCommand) Name() string {
	return s.name
}

// Name	return menu item name string
func (s simpleCommand) Help() string {
	return s.help
}

// Process item action
func (s simpleCommand) Exec(line string) error {
	return s.exec(line)
}

// Compliter get readline compliter
func (s simpleCommand) Compliter() (comp []Compliter) {
	if len(s.compliters) > 0 {
		comp = s.compliters[0]()
	}
	return
}

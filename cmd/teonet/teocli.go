package main

import (
	"flag"
	"fmt"
	"strings"

	"github.com/kirill-scherba/teonet"
	"github.com/kirill-scherba/teonet/cmd/teonet/menu"
)

// NewTeocli create new teonet cli client
func NewTeocli(teo *teonet.Teonet, appShort string) (cli *Teocli, err error) {
	cli = &Teocli{teo: teo}

	// Add commands
	cli.addCommands()

	// Create readline based cli menu and add menu items (commands)
	cli.menu, err = menu.New(appShort)
	if err != nil {
		err = fmt.Errorf("can't create menu, %s", err)
		return
	}
	cli.menu.Add(cli.commands...)
	cli.batch = &Batch{cli.menu}
	cli.alias = newAlias()
	cli.api = newAPI()

	return
}

type Teocli struct {
	commands []menu.Item
	batch    *Batch
	alias    *Alias
	menu     *menu.Menu
	api      *API
	teo      *teonet.Teonet
}

// TeocliCommand common Teocli command structure
type TeocliCommand struct{ *Teocli }

// Command get command by name or nil if not found
func (cli Teocli) Command(name string) interface{} {
	for i := range cli.commands {
		if cli.commands[i].Name() == name {
			return cli.commands[i]
		}
	}
	return nil
}

// Run command line interface menu
func (cli Teocli) Run() {
	cli.menu.Run()
}

// setUsage set flags usage helper function
func (cli Teocli) setUsage(usage string, flags *flag.FlagSet, help ...string) {
	savUsage := flags.Usage
	flags.Usage = func() {
		fmt.Print("usage: " + usage + "\n\n")
		if len(help) > 0 && len(help[0]) > 0 {
			fmt.Print(strings.ToUpper(help[0][0:1]) + help[0][1:] + "\n\n")
		}
		savUsage()
		fmt.Println()
	}
}

// NewFlagSet
func (cli Teocli) NewFlagSet(name, usage string, help ...string) (flags *flag.FlagSet) {
	flags = flag.NewFlagSet(name, flag.ContinueOnError)
	cli.setUsage(name+" "+usage, flags, help...)
	return
}

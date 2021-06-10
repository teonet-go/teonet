// Copyright 2021 Kirill Scherba <kirill@scherba.ru>. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Teonet CLI - teonet command line interface application
// Allow connect to any teonet apllication and send/receive data or commands
package main

import (
	"errors"
	"flag"
	"fmt"
	"time"

	"github.com/kirill-scherba/teonet/cmd/teocli/menu"

	"github.com/kirill-scherba/teonet"
)

const (
	appShort   = "teocli"
	appName    = "Teonet CLI application"
	appVersion = "0.0.1"
)

func main() {

	// Application logo
	teonet.Logo(appName, appVersion)

	// Connect to Teonet
	teo, err := teonet.New(appShort)
	if err != nil {
		fmt.Println("can't init Teonet, error:", err)
		return
	}
	fmt.Println("teonet address", teo.Address())

	// Connect to teonet
	err = teo.Connect()
	if err != nil {
		fmt.Println("can't connect to Teonet, error:", err)
		return
	}
	fmt.Println("connected to teonet")

	// Create Teonet CLI
	cli, err := NewTeocli(teo)
	if err != nil {
		fmt.Println("can't create Teonet CLI, err:", err)
	}

	// Run Teonet CLI commands menu
	fmt.Print(
		"\n",
		"Usage:	<command> [arguments]\n",
		"use help command to get commands list\n\n",
	)
	cli.menu.Run()
}

// NewTeocli create new teonet cli client
func NewTeocli(teo *teonet.Teonet) (cli *Teocli, err error) {
	cli = &Teocli{teo: teo}

	// Add commands
	cli.commands = append(cli.commands,
		cli.newCmdConnectTo(),
		cli.newCmdSendTo(),
	//	b.newCmdMargin(),
	// 	b.newCmdClone(),
	// 	b.newCmdLoan(),
	// 	b.newCmdRepay(),
	// 	b.newCmdOrder(),
	// 	b.newCmdKlines(),
	)

	// Create readline based cli menu and add menu items (commands)
	cli.menu, err = menu.New()
	if err != nil {
		err = fmt.Errorf("can't create menu, %s", err)
		return
	}
	cli.menu.Add(cli.commands...)

	return
}

type Teocli struct {
	commands []menu.Item
	menu     *menu.Menu
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
func (cli Teocli) setUsage(usage string, flags *flag.FlagSet) {
	savUsage := flags.Usage
	flags.Usage = func() { fmt.Print("usage: " + usage + "\n\n"); savUsage(); fmt.Println() }
}

// NewFlagSet
func (cli Teocli) NewFlagSet(name, usage string) (flags *flag.FlagSet) {
	flags = flag.NewFlagSet(name, flag.ContinueOnError)
	cli.setUsage(name+" "+usage, flags)
	return
}

// Command name
const (
	cmdConnectTo = "connectto"
	cmdSendTo    = "sendto"
	// cmdLoan   = "loan"
	// cmdRepay  = "repay"
	// cmdOrder  = "order"
	// cmdKlines = "klines"
	cmdHelp = "help"
)

// Create cmdConnectTo commands
func (cli *Teocli) newCmdConnectTo() menu.Item {
	return CmdConnectTo{TeocliCommand: TeocliCommand{cli}}
}

// CmdConnectTo connect to peer command ----------------------------------------
type CmdConnectTo struct {
	TeocliCommand
}

func (c CmdConnectTo) Name() string  { return cmdConnectTo }
func (c CmdConnectTo) Usage() string { return "<address>" }
func (c CmdConnectTo) Help() string  { return "connect to teonet peer" }
func (c CmdConnectTo) Exec(line string) (err error) {
	flags := c.NewFlagSet(c.Name(), c.Usage())
	var list bool
	flags.BoolVar(&list, "list", list, "list all connected peers")
	err = flags.Parse(c.menu.SplitSpace(line))
	if err != nil {
		return
	}
	args := flags.Args()

	// Check help
	if len(args) > 0 && args[0] == cmdHelp {
		flags.Usage()
		return
	}

	// Check -list flag
	if list {
		peers := c.teo.Peers()
		for i := range peers {
			fmt.Printf("%s\n", peers[i])
		}
		return
	}

	// Check length of arguments
	if len(args) != 1 {
		flags.Usage()
		err = errors.New("wrong number of arguments")
		return
	}

	// Connect to Peer
	var address = args[0]
	err = c.teo.ConnectTo(address)
	// func(c *teonet.Channel, p *teonet.Packet, err error) bool {
	// 	if err != nil {
	// 		fmt.Printf("got error: %s, from: %s\n", err, address)
	// 	}
	// 	fmt.Printf("\ngot '%s' from: %s\n", p.Data(), address)

	// 	return true
	// },

	if err != nil {
		fmt.Printf("can't connect to %s, error: %s\n", address, err)
	}
	fmt.Println("connected to", address)

	// Create API interface and get API
	// _, _ :=
	// c.teo.NewAPIClient(address)
	// if err != nil {
	// 	fmt.Printf("can't api %s, error: %s\n", address, err)
	// 	return
	// }
	// fmt.Println("get api", address)

	return
}
func (c CmdConnectTo) Compliter() (cmpl []menu.Compliter) {
	return c.menu.MakeCompliterFromString([]string{"new", "show"})
}

// Create CmdSendTo commands
func (cli *Teocli) newCmdSendTo() menu.Item {
	return CmdSendTo{TeocliCommand: TeocliCommand{cli}}
}

// CmdSendTo send to peer command ----------------------------------------------
type CmdSendTo struct {
	TeocliCommand
}

func (c CmdSendTo) Name() string  { return cmdSendTo }
func (c CmdSendTo) Usage() string { return "<address> [data]" }
func (c CmdSendTo) Help() string  { return "send data to teonet peer" }
func (c CmdSendTo) Exec(line string) (err error) {
	flags := c.NewFlagSet(c.Name(), c.Usage())
	err = flags.Parse(c.menu.SplitSpace(line))
	if err != nil {
		return
	}
	args := flags.Args()

	// Check help
	if len(args) > 0 && args[0] == cmdHelp {
		flags.Usage()
		return
	}

	// Check length of arguments
	if len(args) == 0 {
		flags.Usage()
		err = errors.New("wrong number of arguments")
		return
	}

	// Address and Data
	var address = args[0]
	var data []byte
	if len(args) > 1 {
		for i, v := range args[1:] {
			if i > 0 {
				data = append(data, byte(' '))
			}
			data = append(data, []byte(v)...)
		}
	}

	// Send data to peer and wait answer
	wait := make(chan interface{})
	id, err := c.teo.SendTo(address, data, func(c *teonet.Channel, p *teonet.Packet, err error) bool {
		if err != nil {
			// fmt.Printf("got error: %s, from: %s\n", err, address)
			return false
		}
		fmt.Printf("got '%s' from: %s\n", p.Data(), address)
		wait <- struct{}{}
		return true
	})
	if err != nil {
		fmt.Printf("can't send to %s, error: %s\n", address, err)
		if err == teonet.ErrPeerNotConnected {
			fmt.Printf("use: 'connectto %s' to connect\n", address)
		}
		return nil
	}
	fmt.Printf("send data to %s, packet ip: %d\n", address, id)
	select {
	case <-wait:
		close(wait)
	case <-time.After(time.Duration(3 * time.Second)):
		fmt.Println("can't got data, error: timeout")
	}

	return
}
func (c CmdSendTo) Compliter() (cmpl []menu.Compliter) {
	return c.menu.MakeCompliterFromString([]string{})
}

package main

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/kirill-scherba/teomon/teomon"
	"github.com/kirill-scherba/teonet"
	"github.com/kirill-scherba/teonet/cmd/teonet/menu"
)

// Command name
const (
	cmdConnectTo = "connectto"
	cmdSendTo    = "sendto"
	cmdAlias     = "alias"
	cmdAPI       = "api"
	cmdHelp      = "help"
)

var ErrWrongNumArguments = errors.New("wrong number of arguments")

// addCommands add commands
func (cli *Teocli) addCommands() {
	cli.commands = append(cli.commands,
		cli.newCmdAlias(),
		cli.newCmdConnectTo(),
		cli.newCmdSendTo(),
		cli.newCmdAPI(),
	)
}

// Create cmdAlias commands
func (cli *Teocli) newCmdAlias() menu.Item {
	return CmdAlias{TeocliCommand: TeocliCommand{cli}}
}

// CmdAlias connect to peer command ----------------------------------------
type CmdAlias struct {
	TeocliCommand
}

func (c CmdAlias) Name() string  { return cmdAlias }
func (c CmdAlias) Usage() string { return "<name> <address>" }
func (c CmdAlias) Help() string  { return "create alias for address" }
func (c CmdAlias) Exec(line string) (err error) {
	var list bool
	flags := c.NewFlagSet(c.Name(), c.Usage(), c.Help())
	flags.BoolVar(&list, "list", list, "list of alias")
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
		aliases := c.alias.list()
		for i := range aliases {
			fmt.Printf("%s\n", aliases[i])
		}
		return
	}

	// Check length of arguments
	if len(args) != 2 {
		flags.Usage()
		err = ErrWrongNumArguments
		return
	}

	// Add alias
	c.alias.add(args[0], args[1])

	return
}
func (c CmdAlias) Compliter() (cmpl []menu.Compliter) {
	return c.menu.MakeCompliterFromString([]string{"-list"})
}

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
	var list bool
	flags := c.NewFlagSet(c.Name(), c.Usage(), c.Help())
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
			alias := c.alias.Name(peers[i])
			if alias != "" {
				alias = " - " + alias
			}
			fmt.Printf("%s\n", peers[i]+alias)
		}
		return
	}

	// Check length of arguments
	if len(args) != 1 {
		flags.Usage()
		err = ErrWrongNumArguments
		return
	}

	// Connect to Peer
	var address = c.alias.Address(args[0])
	err = c.teo.ConnectTo(address)
	if err != nil {
		return
	}
	fmt.Println("connected to", address)

	return
}
func (c CmdConnectTo) Compliter() (cmpl []menu.Compliter) {
	return c.menu.MakeCompliterFromString([]string{"-list"})
}

// Create cmdConnectTo commands
func (cli *Teocli) newCmdAPI() menu.Item {
	return CmdAPI{TeocliCommand: TeocliCommand{cli}}
}

// CmdConnectTo connect to peer command ----------------------------------------
type CmdAPI struct {
	TeocliCommand
}

func (c CmdAPI) Name() string  { return cmdAPI }
func (c CmdAPI) Usage() string { return "<address> [command] [arguments...]" }
func (c CmdAPI) Help() string  { return "get peers api" }
func (c CmdAPI) Exec(line string) (err error) {
	flags := c.NewFlagSet(c.Name(), c.Usage(), c.Help())
	var list bool
	flags.BoolVar(&list, "list", list, "list all connected api")
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
		apis := c.api.list(c.alias)
		for i := range apis {
			fmt.Printf("%s\n", apis[i])
		}
		return
	}

	// Check length of arguments
	if len(args) == 0 {
		flags.Usage()
		err = errors.New("wrong number of arguments")
		return
	}

	// Create API interface and get API
	var address = c.alias.Address(args[0])
	api, ok := c.api.get(address)
	if !ok {
		api, err = c.teo.NewAPIClient(address)
		if err != nil {
			fmt.Printf("can't get api %s, error: %s\n", address, err)
			if err == teonet.ErrPeerNotConnected {
				fmt.Printf("use: 'connectto %s' to connect\n", address)
			}
			return nil
		}
		c.api.add(address, api)
	}
	if len(args) == 1 {
		fmt.Print(api.String() + "\n")
		return
	}

	var command = args[1]
	var data []byte
	if len(args) > 2 {
		for i, v := range args[2:] {
			if i > 0 {
				data = append(data, byte(' '))
			}
			data = append(data, []byte(v)...)
		}
	}
	// Send no answer command
	if answerMode, ok := api.AnswerMode(command); ok && answerMode == teonet.NoAnswer {
		_, err = api.SendTo(command, data)
		if err != nil {
			fmt.Printf("can't send api command %s, error: %s\n", command, err)
			err = nil
		}
		return
	}
	// Send command and wait answer
	wait := make(chan interface{})
	_, err = api.SendTo(command, data, func(data []byte, err error) {
		if err != nil {
			// fmt.Println("got error:", err)
		} else {
			if ret, ok := api.Return(command); ok {
				switch {
				case strings.Contains(ret, "string"):
					fmt.Println("got answer:", string(data))
				case strings.Contains(ret, "[]*Metric"):
					var peers teomon.Peers
					err = peers.UnmarshalBinary(data)
					if err != nil {
						return
					}
					fmt.Println(peers)
				default:
					fmt.Println("got answer:", data)
				}
			}
		}
		wait <- struct{}{}
	})
	if err != nil {
		fmt.Printf("can't send api command %s, error: %s\n", command, err)
		return nil
	}
	select {
	case <-wait:
	case <-time.After(time.Duration(10 * time.Second)):
		fmt.Println("can't got answer, error: timeout")
	}

	return
}
func (c CmdAPI) Compliter() (cmpl []menu.Compliter) {
	return c.menu.MakeCompliterFromString([]string{"-list"})
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
	flags := c.NewFlagSet(c.Name(), c.Usage(), c.Help())
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
	var address = c.alias.Address(args[0])
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
	id, err := c.teo.SendTo(address, data, func(c *teonet.Channel, p *teonet.Packet, e *teonet.Event) bool {
		// if err != nil {
		if e.Event != teonet.EventData {
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

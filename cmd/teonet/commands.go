package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync/atomic"
	"time"

	"github.com/teonet-go/teomon"
	"github.com/teonet-go/teonet"
	"github.com/teonet-go/teonet/cmd/teonet/menu"
)

// Command name
const (
	cmdConnectTo = "connectto"
	cmdSendTo    = "sendto"
	cmdAlias     = "alias"
	cmdStat      = "stat"
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
		cli.newCmdStat(),
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
	var list, save bool
	flags := c.NewFlagSet(c.Name(), c.Usage(), c.Help())
	flags.BoolVar(&list, "list", list, "show list of alias")
	flags.BoolVar(&save, "save", list, "save list of alias")
	err = flags.Parse(c.menu.SplitSpace(line))
	if err != nil {
		return
	}
	args := flags.Args()

	switch {
	// Check help
	case len(args) > 0 && args[0] == cmdHelp:
		flags.Usage()
		return

	// Check -list flag
	case list:
		aliases := c.alias.list()
		for i := range aliases {
			fmt.Printf("%s\n", aliases[i])
		}
		return

	// Check -save flag
	case save:
		aliases := c.alias.list()
		c.batch.Save(aliasBatchFile, cmdAlias, aliases)
		return

	// Check length of arguments
	case len(args) != 2:
		flags.Usage()
		err = ErrWrongNumArguments
		return
	}

	// Add alias
	c.alias.add(args[0], args[1])

	return
}
func (c CmdAlias) Compliter() (cmpl []menu.Compliter) {
	return c.menu.MakeCompliterFromString([]string{"-list", "-save"})
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
	var list, save bool
	flags := c.NewFlagSet(c.Name(), c.Usage(), c.Help())
	flags.BoolVar(&list, "list", list, "list connected peers")
	flags.BoolVar(&save, "save", list, "save connected peer aliases")
	err = flags.Parse(c.menu.SplitSpace(line))
	if err != nil {
		return
	}
	args := flags.Args()

	switch {
	// Check help
	case len(args) > 0 && args[0] == cmdHelp:
		flags.Usage()
		return

	// Check -list flag
	case list:
		peers := c.teo.Peers()
		for i := range peers {
			alias := c.alias.Name(peers[i])
			if alias != "" {
				alias = " - " + alias
			}
			fmt.Printf("%s\n", peers[i]+alias)
		}
		return

	// Check -save flag
	case save:
		var connecttos []string
		peers := c.teo.Peers()
		for i := range peers {
			alias := c.alias.Name(peers[i])
			if alias != "" {
				connecttos = append(connecttos, alias)
			}
		}
		c.batch.Save(connectBatchFile, cmdConnectTo, connecttos)
		return

	// Check length of arguments
	case len(args) != 1:
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
	return c.menu.MakeCompliterFromString([]string{"-list", "-save"})
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

	// Check help
	if args[0] == cmdHelp {
		flags.Usage()
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
	var editmode int32
	var editparam = string(data)
	wait := make(chan interface{})
	_, err = api.SendTo(command, data, func(data []byte, err error) {
		if err != nil {
			// fmt.Println("got error:", err)
		} else {
			if ret, ok := api.Return(command); ok {
				switch {
				case strings.Contains(ret, "string"):
					fmt.Println("got answer:", string(data))
					err = c.edit(api, data, editparam, &editmode)
					if err != nil {
						// fmt.Println("editor, error:", err)
					}
				case strings.Contains(ret, "[]*Metric"):
					var peers = teomon.NewPeers()
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
Wait:
	select {
	case <-wait:
	case <-time.After(time.Duration(10 * time.Second)):
		// Wait forever in edit mode or print timeout error and exit
		if atomic.LoadInt32(&editmode) > 0 {
			goto Wait
		}
		fmt.Println("can't got answer, error: timeout")
	}

	return
}
func (c CmdAPI) Compliter() (cmpl []menu.Compliter) {
	return c.menu.MakeCompliterFromString([]string{"-list"})
}

// edit received data in os editor and save it back if edit mode enable
func (c CmdAPI) edit(api *teonet.APIClient, req []byte, saveparam string, editmode *int32) (err error) {

	// Unmarshal input data
	var v struct {
		Res     interface{} `json:"res"`
		Edit    bool        `json:"edit"`
		SaveCmd string      `json:"savecmd"`
		Err     string      `json:"err"`
	}
	err = json.Unmarshal(req, &v)
	if err != nil {
		return
	}
	if v.Edit {
		// Set edit mode. We use atomic to safe race when editmode check in
		// another goroutine
		atomic.StoreInt32(editmode, 1)
	} else {
		return
	}
	var vv interface{}
	resstr := fmt.Sprintf("%v", v.Res)
	err = json.Unmarshal([]byte(resstr), &vv)
	if err != nil {
		return
	}

	// Make prety json to edit in editor
	data, err := json.MarshalIndent(vv, "", " ")
	if err != nil {
		return
	}

	// Create temp file
	dir := os.TempDir()
	file, err := os.CreateTemp(dir, "*.json")
	if err != nil {
		return
	}
	filename := file.Name()
	defer os.Remove(filename)

	// Write data to temp file
	file.Write(data)
	file.Close()

	// Edit temp file with editor with editor saved in $EDITOR variable
	editor := os.Getenv("EDITOR")
	if len(editor) == 0 {
		fmt.Println("the EDITOR environment varialle does not set, " +
			"please set it and continue edit")
		return
	}
	fmt.Println("run editor:", editor, filename)
	cmd := exec.Command(editor, filename)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	err = cmd.Run()
	if err != nil {
		return
	}

	// Read temp file changed in editor
	var b []byte
	const bufSize = 1024
	buf := make([]byte, bufSize)
	file, err = os.Open(filename)
	if err != nil {
		return
	}
	for {
		n, err := file.Read(buf)
		if err != nil || n == 0 {
			break
		}
		b = append(b, buf[:n]...)
		if n < bufSize {
			break
		}
	}
	file.Close()

	// Compact json
	err = json.Unmarshal(b, &vv)
	if err != nil {
		return
	}
	data, err = json.Marshal(vv)
	fmt.Println("edited json:", string(data))

	// Send command to save edited file
	fmt.Printf("save command: %s %s,%s\n", v.SaveCmd, saveparam, string(data))
	data = []byte(fmt.Sprintf("%s,%s", saveparam, string(data)))
	_, err = api.SendTo(v.SaveCmd, data)
	if err != nil {
		fmt.Printf("can't send api command %s, error: %s\n", v.SaveCmd, err)
		err = nil
	}

	return
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

// Create CmdStat commands
func (cli *Teocli) newCmdStat() menu.Item {
	return CmdStat{TeocliCommand: TeocliCommand{cli}, set: new(bool)}
}

// CmdStat show local stat command ----------------------------------------------
type CmdStat struct {
	TeocliCommand
	set *bool
}

func (c CmdStat) Name() string  { return cmdStat }
func (c CmdStat) Usage() string { return "" }
func (c CmdStat) Help() string  { return "show local statistic on / off" }
func (c CmdStat) Exec(line string) (err error) {
	*c.set = !*c.set
	c.teo.ShowTrudp(*c.set)
	var onoff = "off"
	if *c.set {
		onoff = "on"
	}
	fmt.Println("\rstat", onoff)
	return
}
func (c CmdStat) Compliter() (cmpl []menu.Compliter) {
	return c.menu.MakeCompliterFromString([]string{})
}

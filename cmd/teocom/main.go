// Copyright 2022 Kirill Scherba <kirill@scherba.ru>. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Teonet Command - teonet command line interface application which connect to
// selected teonet peer and execute it command
package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/teonet-go/teomon"
	"github.com/teonet-go/teonet"
)

const (
	appName    = "Teonet command application"
	appShort   = "teocom"
	appVersion = teonet.Version
)

var appStartTime = time.Now()

func main() {
	// Application logo
	teonet.Logo(appName, appVersion)

	// Parse applications flags
	var p struct {
		appShort    string
		port        int
		stat        bool
		hotkey      bool
		showPrivate bool
		sendTo      string
		sendDelay   int
		logLevel    string
		logFilter   string
		monitor     string
	}
	flag.StringVar(&p.appShort, "name", appShort, "application short name")
	flag.IntVar(&p.port, "p", 0, "local port")
	flag.BoolVar(&p.stat, "stat", false, "show trudp statistic")
	flag.BoolVar(&p.hotkey, "hotkey", false, "start hotkey menu")
	flag.BoolVar(&p.showPrivate, "show-private", false, "show private key")
	flag.StringVar(&p.sendTo, "send-to", "", "send messages to address")
	flag.IntVar(&p.sendDelay, "send-delay", 0, "delay between send message in milleseconds")
	flag.StringVar(&p.logLevel, "loglevel", "NONE", "log level")
	flag.StringVar(&p.logFilter, "logfilter", "", "log filter")
	flag.StringVar(&p.monitor, "monitor", "", "monitor address")
	flag.Parse()

	// Start teonet client
	teo, err := teonet.New(p.appShort, p.port, reader, teonet.Stat(p.stat),
		teonet.Hotkey(p.hotkey), p.logLevel, teonet.Logfilter(p.logFilter),
	)
	if err != nil {
		panic("can't init Teonet, error: " + err.Error())
	}

	// Connect to teonet
	for teo.Connect() != nil {
		// teo.Log().Debug.Println("can't connect to Teonet, error:", err)
		time.Sleep(1 * time.Second)
	}
	defer teo.Close()

	// Teonet address
	fmt.Printf("Teonet address: %s\n\n", teo.Address())

	// Show this application private key
	if p.showPrivate {
		fmt.Printf("Teonet private key hex: %x\n", teo.GetPrivateKey())
		os.Exit(0)
	}

	// Check arguments, we are loking at least one argument
	fmt.Println("Application arguments:")
	args := flag.Args()
	if len(args) < 1 {
		fmt.Println("error: arguments mising, need at least one argument")
		os.Exit(1)
	}
	cmd := args[0]
	arg := strings.Join(args[1:], " ")
	fmt.Println("  command:", cmd)
	fmt.Println("  argument:", arg)
	fmt.Println()

	// Check command
	fmt.Println("Application command:")
	cmdType := "command"
	cmds := strings.Split(cmd, ":")
	if len(cmds) < 2 {
		fmt.Println("error: wrong command, need address:cmd_number or address:api:cmd")
		os.Exit(2)
	}
	addr := cmds[0]
	if cmds[1] == "api" {
		cmdType = "api"
		if len(cmds) < 3 {
			fmt.Println("error: wrong command, need address:api:cmd")
			os.Exit(3)
		}
		cmd = cmds[2]
	} else {
		cmd = cmds[1]
	}
	fmt.Println("  type:", cmdType)
	fmt.Println("  address:", addr)
	fmt.Println("  command:", cmd)
	fmt.Println()

	// Connect to monitor
	if len(p.monitor) > 0 {
		teomon.Connect(teo, p.monitor, teomon.Metric{
			AppName:      appName,
			AppShort:     appShort,
			AppVersion:   appVersion,
			TeoVersion:   teonet.Version,
			AppStartTime: appStartTime,
		})
	}

	// Connect to selected peer
	for {
		err = teo.ConnectTo(addr)
		if err == nil {
			break
		}
		fmt.Println("can't connect to teonet peer", addr)
		time.Sleep(5 * time.Second)
	}
	defer teo.CloseTo(addr)
	fmt.Println("Connected to:", addr)

	// Execute command
	c, err := strconv.Atoi(cmd)
	if err != nil {
		fmt.Println("error: wrong command number", cmd)
		os.Exit(4)
	}
	fmt.Println("Send to:", cmd, arg)
	fmt.Println()
	_, err = teo.Command(c, arg).SendTo(addr)
	if err != nil {
		fmt.Printf("error: can't send command %d to teonet peer %s %s\n", c, addr, err)
		os.Exit(5)
	}

	// Wait result
	// This command return data without command, if cmd returned should add
	// cmd to attr WaitFrom parameter
	data, err := teo.WaitFrom(addr)
	if err != nil {
		fmt.Printf("error: can't get command %d answer from teonet peer %s %s\n", c, addr, err)
		os.Exit(6)
	}

	fmt.Println("Got result:")
	fmt.Println(data)
	fmt.Println(string(data))
}

// reader is main application reader it receive and process messages
func reader(teo *teonet.Teonet, c *teonet.Channel, p *teonet.Packet, e *teonet.Event) bool {
	// Skip not Data events
	// if err != nil {
	if e.Event != teonet.EventData {
		return false
	}

	// In server mode
	if c.ServerMode() {

		// Print received message
		teo.Log().Debug.Printf("got from %s, \"%s\", len: %d, id: %d, tt: %6.3fms\n",
			c, p.Data(), len(p.Data()), p.ID(),
			float64(c.Triptime().Microseconds())/1000.0,
		)

		// Send answer
		answer := []byte("Teonet answer to " + string(p.Data()))
		c.Send(answer)
	}

	return true
}

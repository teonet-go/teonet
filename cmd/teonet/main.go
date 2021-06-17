// Copyright 2021 Kirill Scherba <kirill@scherba.ru>. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Teonet CLI - teonet command line interface application
// Allow connect to any teonet apllication and send/receive data or commands
package main

import (
	"fmt"

	"github.com/kirill-scherba/teonet"
)

const (
	appShort   = "teonet"
	appName    = "Teonet CLI application"
	appVersion = "0.2.1"
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

	// Run batch files
	cli.batch.run(aliasBatchFile)
	cli.batch.run(connectBatchFile)

	// Run Teonet CLI commands menu
	fmt.Print(
		"\n",
		"Usage:	<command> [arguments]\n",
		"use help command to get commands list\n\n",
	)
	cli.menu.Run()
}

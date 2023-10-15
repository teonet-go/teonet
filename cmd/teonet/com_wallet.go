// Copyright 2023 Kirill Scherba <kirill@scherba.ru>. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Teonet CLI application Wallet Commands processing module.

package main

type walletCommands struct {
}

const (
	usageCommand = `
usage: api -wallet <address> <command> [arguments...]

Application wallet commands:
  new
        creates new wallet mnemonic (create new wallet)
  insert <mnemonic>
        inserts your previously saved wallet mnemonic (import wallet)
  show
        shows current wallet mnemonic (export wallet)
  master
        shows saved master key (export wallet by master key)
  password <password>
        sets password to save and read mnemonic and master key at this host
  save
        save wallet parameters on this host
  delete
        delete wallet parameters from this host`

	descriptionNew = `New wallet mnemonic created.

The new wallet is running now. If you used another wallet before this command
and it mnemonic was not saved - you lost it. If your previously wallet was
saved on this host but you have not copy it - execute load command:

  api -wallet teos3 load

To show created wallet mnemonic - execute show command:

  api -wallet teos3 show

To save created wallet mnemonic on this host - execute save command:

  api -wallet teos3 seve

`
)

// AppWalletUsage return application walet usage string
func (c walletCommands) AppWalletUsage() string { return c.usage() }

// AppWalletProcess process application walet options
func (c walletCommands) AppWalletProcess(args []string) string {
	return c.process(args)
}

// usage returns application wallet commands usage string
func (c walletCommands) usage() string {
	return usageCommand
}

// process application wallet commands
func (c walletCommands) process(args []string) string {
	switch args[1] {

	case "new":
		return descriptionNew

	case "insert":
		return "wallet inserted"

	case "load":
		return "popa gnom stol"

	case "show":
		return "popa gnom stol"

	case "master":
		return "eb2345ht"

	case "password":
		return "done"

	case "save":
		return "done"

	case "delete":
		return "done"

	default:
		return usageCommand
	}

}

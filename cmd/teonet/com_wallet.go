// Copyright 2023 Kirill Scherba <kirill@scherba.ru>. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Teonet CLI application Wallet Commands processing module.

package main

import (
	"fmt"

	"github.com/teonet-go/teocrypt/mnemonic"
)

// walletCommands contains data and methods to process -wallet commands
type walletCommands struct {
	walletConfig mnemonic.MnemonicConfig
}

// colos returns input text with terminal colors symbols added.
func color(text string) string { return "\033[32m" + text + "\033[0m" }

// AppWalletUsage return application walet usage string
func (c walletCommands) AppWalletUsage() string { return c.usage() }

// AppWalletProcess process application walet options
func (c *walletCommands) AppWalletProcess(appShort string, args []string) string {
	return c.process(appShort, args)
}

// usage returns application wallet commands usage string
func (c walletCommands) usage() string {
	return usageCommand
}

// process application wallet commands
func (c *walletCommands) process(appShort string, args []string) string {
	switch args[1] {

	case "new":
		c.newWallet(appShort)
		return fmt.Sprintf(descriptionNew, appShort)

	case "insert":
		return "wallet inserted"

	case "load":
		return c.loadWallet(appShort)

	case "show":
		return c.showWallet(appShort)

	case "password":
		return "done"

	case "save":
		return c.saveWallet(appShort)

	case "delete":
		return "done"

	default:
		return usageCommand
	}

}

// newWallet creates new wallet.
func (c *walletCommands) newWallet(appShort string) (err error) {

	// Generate new m
	m, err := mnemonic.NewMnemonic()
	if err != nil {
		return
	}
	fmt.Println("mnemonic:", m)
	c.walletConfig.Mnemonic = []byte(m)

	// Generate private and public keys from mnemonic
	privateKey, _, err := mnemonic.GenerateKeys(string(m))
	if err != nil {
		return
	}
	fmt.Println("privateKey:", privateKey)
	c.walletConfig.PrivateKey = []byte(privateKey)

	return
}

// showWallet shows current wallet mnemonic and key
func (c *walletCommands) showWallet(appShort string) string {
	if len(c.walletConfig.Mnemonic) == 0 && len(c.walletConfig.PrivateKey) == 0 {
		return fmt.Sprintf(descriptionShowError, appShort)
	}
	return fmt.Sprintf(descriptionShow, appShort, c.walletConfig.Mnemonic,
		c.walletConfig.PrivateKey)
}

// saveWallet save current wallet mnemonic and key
func (c walletCommands) saveWallet(appShort string) string {

	if err := c.walletConfig.Save(appShort); err != nil {
		return "error during saving: " + err.Error()
	}

	return "Saved."
}

// loadWallet loads previously saved wallet mnemonic and key
func (c *walletCommands) loadWallet(appShort string) string {

	if err := c.walletConfig.Load(appShort); err != nil {
		return "error during loading: " + err.Error()
	}

	return fmt.Sprintf(descriptionLoad, appShort)
}

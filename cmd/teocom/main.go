// Copyright 2022 Kirill Scherba <kirill@scherba.ru>. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Teonet Command - teonet command line interface application which connect to
// selected teonet peer and execute it command
package main

import "github.com/teonet-go/teonet"

const (
	appName    = "Teonet command application"
	appShort   = "teocom"
	appVersion = teonet.Version
)

func main() {
	// Application logo
	teonet.Logo(appName, appVersion)

	// TODO: Parse parameters

	// TODO: Connect to teonet

	// TODO: Connect to selected peer

	// TODO: Execute command

	// TODO: Return result
}

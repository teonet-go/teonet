// Copyright 2021-22 Kirill Scherba <kirill@scherba.ru>. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Teonet connect URL module

package teonet

import "os"

const (
	verURL = "v5"
	proURL = "https://teonet.cloud"
	devURL = "http://dev.myteo.net:10000"
)

// connectURL connect URL struct and method receiver
type connectURL struct {
	authURL, rauthURL, rauthPage string
}

// newConnectURL create new connectURL and make auth connectr URLs
func (teo *Teonet) newConnectURL() {
	teo.connectURL = new(connectURL)
	teo.connectURL.makeURLs()
}

// makeURLs make auth connectr URLs
func (c *connectURL) makeURLs() {
	// make URLs
	const fullDevURL = devURL + "/" + verURL + "/"
	const fullProdURL = proURL + "/" + verURL + "/"
	// auth
	const authPage = "auth"
	const authProdURL = fullProdURL + authPage
	const authDevURL = fullDevURL + authPage
	// rauth
	c.rauthPage = "rauth"
	rauthProdURL := fullProdURL + c.rauthPage
	rauthDevURL := fullDevURL + c.rauthPage
	// auth & rauth depend of TEOENV mode, if TEOENV=dev than dev mode
	if c.devMode() {
		c.authURL = authDevURL
		c.rauthURL = rauthDevURL
	} else {
		c.authURL = authProdURL
		c.rauthURL = rauthProdURL
	}
}

// devMode return true in development mode
func (c *connectURL) devMode() (ok bool) {
	if os.Getenv("TEOENV") == "dev" {
		ok = true
	}
	return
}

// Copyright 2021-2023 Kirill Scherba <kirill@scherba.ru>. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Teonet CLI API Map processing module.

package main

import (
	"sync"

	"github.com/teonet-go/teonet"
)

// API holds map of connected Teonet API clients, mutext to safe this map and
// methods to manage this map.
type API struct {
	m apiMap
	sync.RWMutex
}
type apiMap map[string]*teonet.APIClient

// newAPI creates a new API object
func newAPI() (a *API) {
	a = new(API)
	a.m = make(apiMap)
	return
}

// add adds record to API map
func (a *API) add(address string, api *teonet.APIClient) {
	a.Lock()
	defer a.Unlock()
	a.m[address] = api
}

// del deletes record from API map
func (a *API) del(address string) {
	a.Lock()
	defer a.Unlock()
	delete(a.m, address)
}

// get gets record from API map by name
func (a *API) get(address string) (api *teonet.APIClient, ok bool) {
	a.RLock()
	defer a.RUnlock()
	api, ok = a.m[address]
	return
}

// list returns slice of 'address - name' in API map
func (a *API) list(alias *Alias) (list []string) {
	a.RLock()
	defer a.RUnlock()
	for address := range a.m {
		list = append(list, address+" - "+alias.Name(address))
	}
	return
}

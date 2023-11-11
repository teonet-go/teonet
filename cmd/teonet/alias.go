// Copyright 2021-2023 Kirill Scherba <kirill@scherba.ru>. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Teonet CLI Alias Map processing module.

package main

import "sync"

// Alias holds map of Teonet Peers address aliases, mutext to safe this map and
// methods to manage this map.
type Alias struct {
	m aliasMap
	sync.RWMutex
}

// newAPI creates a new Alias object
func newAlias() (a *Alias) {
	a = new(Alias)
	a.m = make(aliasMap)
	return
}

type aliasMap map[string]string

// Address gets address by name
func (a *Alias) Address(name string) string {
	if address, ok := a.get(name); ok {
		return address
	}
	return name
}

// Name gets name by address
func (a *Alias) Name(address string) string {
	if name, ok := a.find(address); ok {
		return name
	}
	return ""
}

// add adds addres to Alias map by name
func (a *Alias) add(name, address string) {
	a.Lock()
	defer a.Unlock()
	a.m[name] = address
}

// del deletes record from Alias map by name
func (a *Alias) del(name string) {
	a.Lock()
	defer a.Unlock()
	delete(a.m, name)
}

// get gets address from Alias map by name
func (a *Alias) get(name string) (address string, ok bool) {
	a.RLock()
	defer a.RUnlock()
	address, ok = a.m[name]
	return
}

// find finds name in Alias map by address and returns name
func (a *Alias) find(address string) (name string, ok bool) {
	a.RLock()
	defer a.RUnlock()
	for n, addr := range a.m {
		if addr == address {
			ok = true
			name = n
			return
		}
	}
	return
}

// list returns slice of 'name address' from Alias map
func (a *Alias) list() (list []string) {
	a.RLock()
	defer a.RUnlock()
	for name, address := range a.m {
		list = append(list, name+" "+address)
	}
	return
}

package main

import "sync"

func newAlias() (a *Alias) {
	a = new(Alias)
	a.m = make(aliasMap)
	return
}

type Alias struct {
	m aliasMap
	sync.RWMutex
}

type aliasMap map[string]string

func (a *Alias) add(name, address string) {
	a.Lock()
	defer a.Unlock()
	a.m[name] = address
}

func (a *Alias) del(name string) {
	a.Lock()
	defer a.Unlock()
	delete(a.m, name)
}

func (a *Alias) get(name string) (address string, ok bool) {
	a.RLock()
	defer a.RUnlock()
	address, ok = a.m[name]
	return
}

func (a *Alias) list() (list []string) {
	a.RLock()
	defer a.RUnlock()
	for name, address := range a.m {
		list = append(list, name+" "+address)
	}
	return
}

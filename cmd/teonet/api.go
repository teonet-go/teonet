package main

import (
	"sync"

	"github.com/kirill-scherba/teonet"
)

func newAPI() (a *API) {
	a = new(API)
	a.m = make(apiMap)
	return
}

type API struct {
	m apiMap
	sync.RWMutex
}

type apiMap map[string]*teonet.APIClient

// func (a *API) Address(name string) string {
// 	if address, ok := a.get(name); ok {
// 		return address
// 	}
// 	return name
// }

func (a *API) add(name string, api *teonet.APIClient) {
	a.Lock()
	defer a.Unlock()
	a.m[name] = api
}

func (a *API) del(name string) {
	a.Lock()
	defer a.Unlock()
	delete(a.m, name)
}

func (a *API) get(name string) (api *teonet.APIClient, ok bool) {
	a.RLock()
	defer a.RUnlock()
	api, ok = a.m[name]
	return
}

func (a *API) list(alias *Alias) (list []string) {
	a.RLock()
	defer a.RUnlock()
	for name := range a.m {
		list = append(list, name+" - "+alias.Name(name))
	}
	return
}

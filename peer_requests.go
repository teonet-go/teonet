// Copyright 2021 Kirill Scherba <kirill@scherba.ru>. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Peer requests module

package teonet

import (
	"fmt"
	"sync"
	"time"

	"github.com/kirill-scherba/trudp"
)

func (teo *Teonet) newPeerRequests() {
	teo.peerRequests = new(peerRequests)
	teo.peerRequests.m = make(map[string]*peerRequestsData)
	go teo.peerRequests.process()
}

// peerRequests holder
type peerRequests struct {
	m map[string]*peerRequestsData
	sync.RWMutex
}

type peerRequestsData struct {
	*ConnectToData
	time.Time
}

func (p *peerRequests) add(con *ConnectToData) {
	p.Lock()
	defer p.Unlock()
	p.m[con.ID] = &peerRequestsData{con, time.Now()}
	fmt.Println("peer request add, id:", con.ID)
}

func (p *peerRequests) del(id string) {
	p.Lock()
	defer p.Unlock()
	delete(p.m, id)
	fmt.Println("peer request del, id:", id)
}

func (p *peerRequests) get(id string) (res *peerRequestsData, ok bool) {
	p.RLock()
	defer p.RUnlock()
	res, ok = p.m[id]
	fmt.Println("peer request get, id:", id, ok)
	return
}

func (p *peerRequests) removeDummy() {
	p.RLock()
	for id, rec := range p.m {
		if time.Since(rec.Time) > trudp.ClientConnectTimeout {
			p.RUnlock()
			p.del(id)
			fmt.Println("peer request removed dummy, id:", id)
			p.removeDummy()
			return
		}
	}
	p.RUnlock()
}

func (p *peerRequests) process() {
	for {
		time.Sleep(trudp.ClientConnectTimeout)
		p.removeDummy()
	}
}

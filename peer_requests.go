// Copyright 2021 Kirill Scherba <kirill@scherba.ru>. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Peer requests module

package teonet

import (
	"sync"
	"time"

	"github.com/kirill-scherba/trudp"
)

func (teo *Teonet) newPeerRequests() {
	teo.peerRequests = teo.newConnectRequests()
}

func (teo *Teonet) newConnRequests() {
	teo.connRequests = teo.newConnectRequests()
}

func (teo Teonet) newConnectRequests() *connectRequests {
	c := new(connectRequests)
	c.m = make(map[string]*connectRequestsData)
	go c.process()
	return c
}

// connectRequests holder
type connectRequests struct {
	m map[string]*connectRequestsData
	sync.RWMutex
}

type connectRequestsData struct {
	*ConnectToData
	*chanWait
	time.Time
}

type chanWait chan []byte

func (p *connectRequests) add(con *ConnectToData, waits ...*chanWait) {
	var wait *chanWait
	if len(waits) > 0 {
		wait = waits[0]
	}
	p.Lock()
	defer p.Unlock()
	p.m[con.ID] = &connectRequestsData{con, wait, time.Now()}
	// fmt.Println("connect request add, id:", con.ID)
}

func (p *connectRequests) del(id string) {
	p.Lock()
	defer p.Unlock()
	delete(p.m, id)
	// fmt.Println("connect request del, id:", id)
}

func (p *connectRequests) get(id string) (res *connectRequestsData, ok bool) {
	p.RLock()
	defer p.RUnlock()
	res, ok = p.m[id]
	// fmt.Println("connect request get, id:", id, ok)
	return
}

func (p *connectRequests) removeDummy() {
	p.RLock()
	for id, rec := range p.m {
		if time.Since(rec.Time) > trudp.ClientConnectTimeout {
			p.RUnlock()
			p.del(id)
			// fmt.Println("connect request removed dummy, id:", id)
			p.removeDummy()
			return
		}
	}
	p.RUnlock()
}

func (p *connectRequests) process() {
	for {
		time.Sleep(trudp.ClientConnectTimeout)
		p.removeDummy()
	}
}

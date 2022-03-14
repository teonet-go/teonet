// Copyright 2021 Kirill Scherba <kirill@scherba.ru>. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Teonet connect requests module

package teonet

import (
	"sync"
	"time"

	"github.com/kirill-scherba/tru"
)

// Struct and methods receiver
type connectRequests struct {
	m map[string]*connectRequestsData
	sync.RWMutex
}

// Connect request data
type connectRequestsData struct {
	*ConnectToData
	*chanWait
	time.Time
}

// Wait connect result channel
type chanWait chan []byte

// Check if channel is open (is not closed)
func (c chanWait) IsOpen() (ok bool) {
	ok = true
	select {
	case _, ok = <-c:
	default:
	}
	return
}

// newPeerRequests creates new peer request object
func (teo *Teonet) newPeerRequests() {
	teo.peerRequests = teo.newConnectRequests()
}

// newConnRequests creates new connection request object
func (teo *Teonet) newConnRequests() {
	teo.connRequests = teo.newConnectRequests()
}

// newConnectRequests creates new connect request object
func (teo Teonet) newConnectRequests() *connectRequests {
	c := new(connectRequests)
	c.m = make(map[string]*connectRequestsData)
	go c.process()
	return c
}

// add connect request
func (p *connectRequests) add(con *ConnectToData, waits ...*chanWait) {
	var wait *chanWait
	if len(waits) > 0 {
		wait = waits[0]
	}
	p.Lock()
	defer p.Unlock()
	p.m[con.ID] = &connectRequestsData{con, wait, time.Now()}
}

// del connect request by id
func (p *connectRequests) del(id string) {
	p.Lock()
	defer p.Unlock()
	delete(p.m, id)
}

// get connect request by id
func (p *connectRequests) get(id string) (res *connectRequestsData, ok bool) {
	p.RLock()
	defer p.RUnlock()
	res, ok = p.m[id]
	return
}

// removeDummy remove dummy requests
func (p *connectRequests) removeDummy() {
	p.RLock()
	for id, rec := range p.m {
		if time.Since(rec.Time) > tru.ClientConnectTimeout {
			p.RUnlock()
			p.del(id)
			p.removeDummy()
			return
		}
	}
	p.RUnlock()
}

// process periodically remove dummy requests
func (p *connectRequests) process() {
	for {
		time.Sleep(tru.ClientConnectTimeout)
		p.removeDummy()
	}
}

// Copyright 2021-22 Kirill Scherba <kirill@scherba.ru>. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Teonet channels module

package teonet

import (
	"sync"

	"github.com/kirill-scherba/tru"
)

// channels struct and receiver
type channels struct {
	m_addr map[string]*Channel
	m_chan map[*tru.Channel]*Channel
	auth   *Channel
	tru    *tru.Tru
	teo    *Teonet
	sync.RWMutex
}

// newChannels create channels
func (teo *Teonet) newChannels() {
	teo.channels = new(channels)
	teo.channels.teo = teo
	teo.channels.tru = teo.tru
	if teo.tru == nil {
		panic("tru should be Init befor call to newChannels()")
	}
	teo.channels.m_addr = make(map[string]*Channel)
	teo.channels.m_chan = make(map[*tru.Channel]*Channel)
}

// add new teonet channel
func (c *channels) add(channel *Channel) {
	// remove existing channel with same address
	if ch, ok := c.get(channel.a); ok {
		// TODO: check and remove this comment
		// If new channel used the same tru channel as existing than does not
		// delete tru channel. The c.del function delete tru channel by
		// default
		// var delTrudp bool
		// if ch.c.Addr().String() != channel.c.Addr().String() {
		// 	delTrudp = true
		// }
		// c.del(ch, delTrudp)
		c.del(ch, false)
	}
	c.Lock()
	defer c.Unlock()

	c.m_addr[channel.a] = channel
	c.m_chan[channel.c] = channel

	// Connected - show log message and send Event to main reader
	log.Connect.Println("peer connected:", channel.a)
}

// del delete teonet channel if second parameter omitted or true, the tru
// channel will also deleted
func (c *channels) del(channel *Channel, delTrudps ...bool) {
	var delTrudp = true
	if len(delTrudps) > 0 {
		delTrudp = delTrudps[0]
	}
	c.Lock()
	defer c.Unlock()

	delete(c.m_addr, channel.a)
	delete(c.m_chan, channel.c)
	// TODO: look why channel.c may be nil here
	if delTrudp && channel.c != nil {
		channel.c.Close()
	}
	c.teo.subscribers.del(channel)
	log.Connect.Println("peer disconnected:", channel.a)
}

// get channel by teonet address or by tru channel
func (c *channels) get(attr interface{}) (ch *Channel, exists bool) {
	c.RLock()
	defer c.RUnlock()
	switch v := attr.(type) {
	case string:
		ch, exists = c.m_addr[v]
	case *tru.Channel:
		ch, exists = c.m_chan[v]
	}
	return
}

// get channel by ip:port address
func (c *channels) getByIP(ipport string) (ch *Channel, exists bool) {
	c.RLock()
	defer c.RUnlock()
	for _, v := range c.m_addr {
		if v.c.Addr().String() == ipport {
			ch = v
			exists = true
			break
		}
	}
	return
}

// list get list of channels IPs in nodes struct
func (c *channels) list() (n *nodes) {
	c.RLock()
	defer c.RUnlock()

	n = new(nodes)
	for _, v := range c.m_addr {
		n.address = append(n.address, NodeAddr{
			v.Channel().IP().String(),
			uint32(v.Channel().Port()),
		})
	}
	return
}

// peers get slice of channels address
func (c *channels) peers() (p []string) {
	c.RLock()
	defer c.RUnlock()

	for key := range c.m_addr {
		p = append(p, key)
	}
	return
}

// Peers get slice of channels address
func (teo Teonet) Peers() (p []string) {
	return teo.channels.peers()
}

// Nodes get list of channels IPs in nodes struct
func (teo Teonet) Nodes(attr ...NodeAddr) (n *nodes) {
	if len(attr) == 0 {
		return teo.channels.list()
	}
	n = new(nodes)
	for i := range attr {
		n.address = append(n.address, attr[i])
	}
	return
}

// NumPeers return number of connected peers
func (teo Teonet) NumPeers() int {
	return len(teo.Peers())
}

// setAuth set Auth channel
func (teo *Teonet) setAuth(ch *Channel) {
	teo.channels.Lock()
	defer teo.channels.Unlock()

	teo.channels.auth = ch
}

// getAuth get Auth channel
func (teo Teonet) getAuth() (ch *Channel) {
	teo.channels.RLock()
	defer teo.channels.RUnlock()

	return teo.channels.auth
}

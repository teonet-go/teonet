// Copyright 2019-2022 Kirill Scherba <kirill@scherba.ru>. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Teonet Client Readers module

package teonet

import "sync"

type clientReaders struct {
	clientReaders []Treceivecb // client readers slice
	sync.RWMutex               // mutex
}

// newClientReaders create new clientReaders
func (teo *Teonet) newClientReaders() {
	teo.clientReaders = new(clientReaders)
}

// AddReader add teonet client reader
func (teo Teonet) AddReader(reader TreceivecbShort) {
	teo.clientReaders.addShort(reader)
}

// add teonet client reader
func (c *clientReaders) add(reader Treceivecb) {
	c.Lock()
	defer c.Unlock()
	c.clientReaders = append(c.clientReaders, reader)
}

// addShort add teonet client short reader
func (c *clientReaders) addShort(reader TreceivecbShort) {
	c.add(func(teo *Teonet, c *Channel, p *Packet, e *Event) bool {
		return reader(c, p, e)
	})
}

// send to client readers (to reader from teonet.Init)
func (c *clientReaders) send(teo *Teonet, ch *Channel, p *Packet, e *Event) bool {
	c.RLock()
	for i := range c.clientReaders {
		if c.clientReaders[i] != nil {
			c.RUnlock()
			if c.clientReaders[i](teo, ch, p, e) {
				return true
			}
			c.RLock()
		}
	}
	c.RUnlock()
	return false
}

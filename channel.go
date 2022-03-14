// Copyright 2021-22 Kirill Scherba <kirill@scherba.ru>. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Teonet channel module

package teonet

import (
	"bytes"
	"strings"
	"time"

	"github.com/kirill-scherba/tru"
)

const (
	newChannelPrefix    = "new-"
	newConnectionPrefix = "conn-"
	addressLen          = 35
)

// Channel stract and method receiver
type Channel struct {
	a string       // Teonet address
	c *tru.Channel // Tru channel
}

// new create new teonet channel
func (c *channels) new(channel *tru.Channel) *Channel {
	address := newChannelPrefix + tru.RandomString(addressLen-len(newChannelPrefix))
	return &Channel{address, channel}
}

// Channel get teonet channel by address
func (teo Teonet) Channel(addr string) (ch *Channel, exists bool) {
	return teo.channels.get(addr)
}

// Channel get teonet channel by ip address
func (teo Teonet) ChannelByIP(addr string) (ch *Channel, exists bool) {
	return teo.channels.getByIP(addr)
}

// ServerMode return true if channel in server mode
func (c Channel) ServerMode() bool {
	return c.c.ServerMode()
}

// ClientMode return true if channel in client mode
func (c Channel) ClientMode() bool {
	return !c.c.ServerMode()
}

// Triptime return channels triptime
func (c Channel) Triptime() time.Duration {
	return c.c.Triptime()
}

// Send data to channel
func (c Channel) Send(data []byte, attr ...interface{}) (id int, err error) {
	var delivery = c.checkSendAttr(attr...)
	return c.c.WriteTo(data, delivery)
}

// SendNoWait (or SendDirect) send data to channel, it use inside readers when packet just read
// and resend in quck time. If you send from routine use Send function
func (c Channel) SendNoWait(data []byte, attr ...interface{}) (id int, err error) {
	return c.Send(data, attr)
}

// checkSendAttr check Send function attributes:
// return delevery calback 'func(p *tru.Packet, err error)' and make
// subscribe to answer with callback 'func(c *Channel, p *Packet, e *Event) bool'
func (c Channel) checkSendAttr(attr ...interface{}) (delivery func(p *tru.Packet, err error)) {
	var teo *Teonet
	for i := range attr {
		switch v := attr[i].(type) {

		// Packet delivery callback
		case func(p *tru.Packet, err error):
			delivery = v

		// Teonet
		case *Teonet:
			teo = v

		// Answer callback
		case func(c *Channel, p *Packet, e *Event) bool:
			if teo != nil {
				c.subscribeToAnswer(teo, v)
			}
		}
	}
	return
}

// subscribeToAnswer subscribe to channel answer
func (c Channel) subscribeToAnswer(teo *Teonet, f func(c *Channel, p *Packet, e *Event) bool) (scr *subscribeData, err error) {
	scr, err = teo.Subscribe(c.a, func(c *Channel, p *Packet, e *Event) bool {
		if f(c, p, e) {
			teo.Unsubscribe(scr)
			return true
		}
		return false
	})
	if err != nil {
		return
	}
	return
}

// String is channel stringify and return string with channel address
func (c Channel) String() string {
	if c.a == "" {
		return c.c.Addr().String()
	}
	return c.a
}

// Address eturn string with channel address
func (c Channel) Address() string {
	return c.a
}

// Channel return return poiner to tru channel
func (c Channel) Channel() *tru.Channel {
	return c.c
}

// IsNew return true if channel has 'new' prefix
func (c Channel) IsNew() bool {
	return strings.HasPrefix(c.Address(), newChannelPrefix)
}

// IsConn return true if channel has 'connect' prefix
func (c Channel) IsConn(data []byte) bool {
	return bytes.HasPrefix(data, []byte(newConnectionPrefix))
}

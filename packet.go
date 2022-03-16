// Copyright 2021-22 Kirill Scherba <kirill@scherba.ru>. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Teonet packet module

package teonet

import "github.com/kirill-scherba/tru"

// Packet is teonet Packet structure and methods receiver
type Packet struct {
	*tru.Packet
	from        string
	commandMode bool
}

// From return packets from address
func (p Packet) From() string {
	return p.from
}

// Cmd return packets command number
func (p Packet) Cmd() byte {
	if p.commandMode {
		return p.Packet.Data()[0]
	}
	return 0
}

// Data return packets data
func (p Packet) Data() []byte {
	if p.commandMode {
		return p.Packet.Data()[1:]
	}
	return p.Packet.Data()
}

// setCommandMode set packet command mode, that mean that packet contain
// command + data
func (p *Packet) setCommandMode() *Packet {
	p.commandMode = true
	return p
}

package teonet

import "github.com/kirill-scherba/tru"

// Packet is teonet Packet
type Packet struct {
	*tru.Packet
	from        string
	commandMode bool
}

func (p Packet) From() string {
	return p.from
}

func (p Packet) Cmd() byte {
	if p.commandMode {
		return p.Packet.Data()[0]
	}
	return 0
}

func (p Packet) Data() []byte {
	if p.commandMode {
		return p.Packet.Data()[1:]
	}
	return p.Packet.Data()
}

func (p Packet) RemoveTrailingZero(data []byte) []byte {
	return data
}

func (p *Packet) setCommandMode() *Packet {
	p.commandMode = true
	return p
}

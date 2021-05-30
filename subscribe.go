package teonet

import (
	"errors"

	"github.com/kirill-scherba/trudp"
)

// Subscribe to receive packets from address
func (teo Teonet) Subscribe(address string, reader Treceivecb) (res *subscribeData, err error) {
	c, ok := teo.channels.get(address)
	if !ok {
		err = errors.New("address does not connected")
		return
	}
	teo.subscribe(c, reader)
	return
}

// subscribe to channel data
func (teo Teonet) subscribe(c *Channel, reader Treceivecb) *subscribeData {
	return teo.subscribers.add(c, reader)
}

// unsubscribe from channel data
func (teo Teonet) unsubscribe(s *subscribeData) {
	teo.subscribers.del(s)
}

type subscribeData struct {
	channel *Channel
	reader  Treceivecb
}

type Treceivecb func(teo *Teonet, c *Channel, p *Packet, err error) bool

// Packet is teonet Packet
type Packet struct {
	*trudp.Packet
	from        string
	commandMode bool
}

func (p Packet) From() string {
	return p.from
}

func (p Packet) Cmd() byte {
	if p.commandMode {
		return p.Packet.Data[0]
	}
	return 0
}

func (p Packet) Data() []byte {
	if p.commandMode {
		return p.Packet.Data[1:]
	}
	return p.Packet.Data
}

func (p Packet) RemoveTrailingZero(data []byte) []byte {
	return data
}

func (p *Packet) SetCommandMode() *Packet {
	p.commandMode = true
	return p
}

// newSubscribers create new subscribers (subscribersData)
func (teo *Teonet) newSubscribers() {
	teo.subscribers = new(subscribers)
}

type subscribers []*subscribeData

func (s *subscribers) add(channel *Channel, reader Treceivecb) (res *subscribeData) {
	res = &subscribeData{channel, reader}
	*s = append(*s, res)
	return
}

func (s subscribers) del(subs *subscribeData) {
	for i := range s {
		if s[i] == subs {
			s[i] = nil
		}
	}
}

// send teonet packet to subscribers and return true if message processed
func (s subscribers) send(teo *Teonet, c *Channel, p *Packet, err error) bool {
	for i := range s {
		if s[i] == nil {
			continue
		}
		if s[i].channel == c {
			// fmt.Println("subscribersData got channel")
			if s[i].reader(teo, c, p, err) {
				return true
			}
		}
	}
	return false
}

package teonet

import (
	"github.com/kirill-scherba/trudp"
)

// unsubscribe from channel data
func (teo Teonet) unsubscribe(s *subscribeData) {
	teo.subscribers.del(s)
}

// subscribe to channel data
func (teo Teonet) subscribe(c *Channel, reader Treceivecb) *subscribeData {
	return teo.subscribers.add(c, reader)
}

type subscribeData struct {
	channel *Channel
	reader  Treceivecb
}

type Treceivecb func(teo *Teonet, c *Channel, p *Packet, err error) bool
type Packet struct{ *trudp.Packet }

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

package teonet

import (
	"errors"
)

// Subscribe to receive packets from address. The reader attribute may be
// teonet.Treceivecb or teonet.TreceivecbShort type
func (teo Teonet) Subscribe(address string, reader interface{}) (res *subscribeData, err error) {
	c, ok := teo.channels.get(address)
	if !ok {
		err = errors.New("address does not connected")
		return
	}
	teo.subscribe(c, reader)
	return
}

// subscribe to channel data
func (teo Teonet) subscribe(c *Channel, readerI interface{}) *subscribeData {
	var reader Treceivecb
	switch v := readerI.(type) {
	// case Treceivecb:
	case func(teo *Teonet, c *Channel, p *Packet, err error) bool:
		reader = v
	// case TreceivecbShort:
	case func(c *Channel, p *Packet, err error) bool:
		reader = func(teo *Teonet, c *Channel, p *Packet, err error) bool {
			return v(c, p, err)
		}
	default:
		teo.Log().Panicf("wrong attribute type %T", v)
	}
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

// delete from subscribers by subscribeData or by channel (by channel remove
// all subscibers to channel)
func (s subscribers) del(subs interface{}) {

	switch v := subs.(type) {
	case *subscribeData:
		for i := range s {
			if s[i] == v {
				s[i] = nil
			}
		}

	case *Channel:
		for i := range s {
			if s[i].channel == v {
				s[i] = nil
			}
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

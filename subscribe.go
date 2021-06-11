package teonet

import (
	"container/list"
	"fmt"
	"sync"
)

// Subscribe to receive packets from address. The reader attribute may be
// teonet.Treceivecb or teonet.TreceivecbShort type
func (teo Teonet) Subscribe(address string, reader interface{}) (scr *subscribeData, err error) {
	c, ok := teo.channels.get(address)
	if !ok {
		err = ErrPeerNotConnected
		return
	}
	scr = teo.subscribe(c, reader)
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
		panic(fmt.Sprintf("wrong attribute type %T", v))
	}
	return teo.subscribers.add(c, reader)
}

// Unsubscribe from channel data
func (teo Teonet) Unsubscribe(s *subscribeData) {
	teo.subscribers.del(s)
}

type subscribeData struct {
	channel *Channel
	reader  Treceivecb
}

// newSubscribers create new subscribers (subscribersData)
func (teo *Teonet) newSubscribers() {
	s := new(subscribers)
	s.idx = make(listIdx)
	teo.subscribers = s
}

type subscribers struct {
	lst          list.List // list
	idx          listIdx   // list index by *subscribeData
	sync.RWMutex           // mutex
}
type listIdx map[*subscribeData]*list.Element

func (s *subscribers) add(channel *Channel, reader Treceivecb) (scr *subscribeData) {
	s.Lock()
	defer s.Unlock()

	scr = &subscribeData{channel, reader}
	s.idx[scr] = s.lst.PushBack(scr)
	return
}

func (s *subscribers) del(subs interface{}) {
	s.Lock()
	defer s.Unlock()

	switch v := subs.(type) {
	case *subscribeData:
		if e, ok := s.idx[v]; ok {
			delete(s.idx, v)
			s.lst.Remove(e)
		}

	case *Channel:
		var next *list.Element
		for e := s.lst.Front(); e != nil; e = next {
			next = e.Next()
			scr := e.Value.(*subscribeData)
			if scr.channel == v {
				delete(s.idx, scr)
				s.lst.Remove(e)
			}
		}
	}
}

func (s *subscribers) send(teo *Teonet, c *Channel, p *Packet, err error) bool {
	s.RLock()
	// defer s.RUnlock()

	var next *list.Element
	for e := s.lst.Front(); e != nil; e = next {
		next = e.Next()
		scr := e.Value.(*subscribeData)
		if scr.channel == c {
			s.RUnlock()
			if scr.reader(teo, c, p, err) {
				return true
			}
			s.RLock()
		}
	}

	s.RUnlock()
	return false
}

/*
type subscribers0 []*subscribeData

func (s *subscribers0) add(channel *Channel, reader Treceivecb) (res *subscribeData) {
	res = &subscribeData{channel, reader}
	*s = append(*s, res)
	return
}

// delete from subscribers by subscribeData or by channel (by channel remove
// all subscibers to channel)
// TODO: remove nil slice member aniware or add mutex and don't use nil as delete
func (s subscribers0) del(subs interface{}) {

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
func (s subscribers0) send(teo *Teonet, c *Channel, p *Packet, err error) bool {
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
*/

// Copyright 2021-22 Kirill Scherba <kirill@scherba.ru>. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Teonet subscribe to receive packets module

package teonet

import (
	"container/list"
	"fmt"
	"sync"
)

// List of subscribers with idex by subscribeData and methods receiver
type subscribers struct {
	lst          list.List // list
	idx          listIdx   // list index by *subscribeData
	sync.RWMutex           // mutex
}
type listIdx map[*subscribeData]*list.Element

// subscribeData contain subscribe data
type subscribeData struct {
	channel *Channel
	reader  Treceivecb
}

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

// Unsubscribe from channel data
func (teo Teonet) Unsubscribe(s *subscribeData) {
	teo.subscribers.del(s)
}

// SubscribersNum return number of subscribers
func (teo Teonet) SubscribersNum() int {
	return teo.subscribers.len()
}

// subscribe to channel data
func (teo Teonet) subscribe(c *Channel, readerI interface{}) *subscribeData {
	var reader Treceivecb
	switch v := readerI.(type) {
	// case Treceivecb:
	case func(teo *Teonet, c *Channel, p *Packet, e *Event) bool:
		reader = v
	// case TreceivecbShort:
	case func(c *Channel, p *Packet, e *Event) bool:
		reader = func(teo *Teonet, c *Channel, p *Packet, e *Event) bool {
			return v(c, p, e)
		}
	default:
		panic(fmt.Sprintf("wrong attribute type %T", v))
	}
	return teo.subscribers.add(c, reader)
}

// newSubscribers create new subscribers (subscribersData)
func (teo *Teonet) newSubscribers() {
	s := new(subscribers)
	s.idx = make(listIdx)
	teo.subscribers = s
}

// add subscriber
func (s *subscribers) add(channel *Channel, reader Treceivecb) (scr *subscribeData) {
	s.Lock()
	defer s.Unlock()

	scr = &subscribeData{channel, reader}
	s.idx[scr] = s.lst.PushBack(scr)
	return
}

// del subscriber
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

// len return number of subscribers
func (s *subscribers) len() int {
	s.RLock()
	defer s.RUnlock()
	return len(s.idx)
}

// send packet to all subscribers
func (s *subscribers) send(teo *Teonet, c *Channel, p *Packet, e *Event) bool {
	s.RLock()

	var next *list.Element
	for el := s.lst.Front(); el != nil; el = next {
		next = el.Next()
		scr := el.Value.(*subscribeData)
		if scr.channel == c {
			s.RUnlock()
			if scr.reader(teo, c, p, e) {
				return true
			}
			s.RLock()
		}
	}

	s.RUnlock()
	return false
}

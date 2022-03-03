package teonet

import (
	"bytes"
	"strings"
	"sync"
	"time"

	"github.com/kirill-scherba/tru"
)

const (
	newChannelPrefix    = "new-"
	newConnectionPrefix = "conn-"
	addressLen          = 35
)

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

// channels holder
type channels struct {
	m_addr map[string]*Channel
	m_chan map[*tru.Channel]*Channel
	tru    *tru.Tru
	teo    *Teonet
	sync.RWMutex
}

func (c *channels) add(channel *Channel) {
	// remove existing channel with same address
	if ch, ok := c.get(channel.a); ok {
		// If new channel used the same tru channel as existing than does not
		// delete tru channel. The c.del function delete tru channel by
		// default
		var delTrudp bool
		if ch.c.Addr().String() != channel.c.Addr().String() {
			delTrudp = true
		}
		c.del(ch, delTrudp)
	}
	c.Lock()
	defer c.Unlock()
	c.m_addr[channel.a] = channel
	c.m_chan[channel.c] = channel

	// Connected - show log message and send Event to main reader
	log.Connect.Println("Peer", "connected:", channel.a)
	// go reader(c.teo, channel, nil, &Event{EventConnected, nil})
}

// del channel
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
	log.Connect.Println("Peer", "disconnec:", channel.a)
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

type Channel struct {
	a string       // Teonet address
	c *tru.Channel // Tru channel
}

func (c Channel) ServerMode() bool {
	return c.c.ServerMode()
}

func (c Channel) ClientMode() bool {
	return !c.c.ServerMode()
}

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

func (c Channel) String() string {
	if c.a == "" {
		return c.c.Addr().String()
	}
	return c.a
}

func (c Channel) Address() string {
	return c.a
}

func (c Channel) Channel() *tru.Channel {
	return c.c
}

func (c Channel) IsNew() bool {
	return strings.HasPrefix(c.Address(), newChannelPrefix)
}

func (c Channel) IsConn(data []byte) bool {
	return bytes.HasPrefix(data, []byte(newConnectionPrefix))
}

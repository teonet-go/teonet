package teonet

import (
	"bytes"
	"strings"
	"sync"
	"time"

	"github.com/kirill-scherba/teonet-go/teolog/teolog"
	"github.com/kirill-scherba/trudp"
)

const (
	newChannelPrefix    = "new-"
	newConnectionPrefix = "conn-"
	addressLen          = 35
)

func (teo *Teonet) newChannels() {
	teo.channels = new(channels)
	teo.channels.teo = teo
	teo.channels.trudp = teo.trudp
	if teo.trudp == nil {
		panic("trudp should be Init befor call to newChannels()")
	}
	teo.channels.m_addr = make(map[string]*Channel)
	teo.channels.m_chan = make(map[*trudp.Channel]*Channel)
}

// channels holder
type channels struct {
	m_addr map[string]*Channel
	m_chan map[*trudp.Channel]*Channel
	trudp  *trudp.Trudp
	teo    *Teonet
	sync.RWMutex
}

func (c *channels) add(channel *Channel) {
	// remove existing channel with same address
	if ch, ok := c.get(channel.a); ok {
		// If new channel used the same trudp channel as existing than does not
		// delete trudp channel. The c.del function delete trudp channel by
		// default
		var delTrudp bool
		if ch.c.String() != channel.c.String() {
			delTrudp = true
		}
		c.del(ch, delTrudp)
	}
	c.Lock()
	defer c.Unlock()
	c.m_addr[channel.a] = channel
	c.m_chan[channel.c] = channel

	// Connected - show log message and send Event to main reader
	teolog.Log(teolog.CONNECT, "Peer", "connected:", channel.a)
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
	if delTrudp {
		c.trudp.ChannelDel(channel.c)
	}
	c.teo.subscribers.del(channel)
	teolog.Log(teolog.CONNECT, "Peer", "disconnec:", channel.a)
}

// get channel by teonet address or by trudp channel
func (c *channels) get(attr interface{}) (ch *Channel, exists bool) {
	c.RLock()
	defer c.RUnlock()
	switch v := attr.(type) {
	case string:
		ch, exists = c.m_addr[v]
	case *trudp.Channel:
		ch, exists = c.m_chan[v]
	}
	return
}

// get channel by ip address
func (c *channels) getByIP(ip string) (ch *Channel, exists bool) {
	c.RLock()
	defer c.RUnlock()
	for _, v := range c.m_addr {
		if v.c.String() == ip {
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
			v.Channel().UDPAddr.IP.String(),
			uint32(v.Channel().UDPAddr.Port),
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
func (c *channels) new(channel *trudp.Channel) *Channel {
	address := newChannelPrefix + trudp.RandomString(addressLen-len(newChannelPrefix))
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
	a string         // Teonet address
	c *trudp.Channel // Trudp channel
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

// Send send data to channel
func (c Channel) Send(data []byte, attr ...interface{}) (id uint32, err error) {
	var delivered = c.checkSendAttr(attr...)
	return c.c.Send(data, delivered)
}

// SendNoWait (or SendDirect) send data to channel, it use inside readers when packet just read
// and resend in quck time. If you send from routine use Send function
func (c Channel) SendNoWait(data []byte, attr ...interface{}) (id uint32, err error) {
	var delivered = c.checkSendAttr(attr)
	return c.c.SendNoWait(data, delivered)
}

// checkSendAttr check Send function attributes:
// return delevered calback 'func(p *trudp.Packet)' and
// subscribe to answer with callback 'func(c *Channel, p *Packet, e *Event) bool'
func (c Channel) checkSendAttr(attr ...interface{}) (delivered func(p *trudp.Packet)) {
	var teo *Teonet
	for i := range attr {
		switch v := attr[i].(type) {

		// Packet delivered callback
		case func(p *trudp.Packet):
			delivered = v

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
		return c.c.String()
	}
	return c.a
}

func (c Channel) Address() string {
	return c.a
}

func (c Channel) Channel() *trudp.Channel {
	return c.c
}

func (c Channel) IsNew() bool {
	return strings.HasPrefix(c.Address(), newChannelPrefix)
}

func (c Channel) IsConn(data []byte) bool {
	return bytes.HasPrefix(data, []byte(newConnectionPrefix))
}

package teonet

import (
	"sync"
	"time"

	"github.com/kirill-scherba/trudp"
)

const (
	newChannelPrefix = "new-"
	addressLen       = 35
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
		c.del(ch)
		// TODO: what is this??? :-)
		// if ch.c.String() != channel.c.String() {
		// 	c.trudp.ChannelDel(ch.c)
		// }
	}

	// add new channel
	c.Lock()
	defer c.Unlock()
	c.m_addr[channel.a] = channel
	c.m_chan[channel.c] = channel
	c.teo.log.Println("client connected:", channel.a)
}

// del channel
func (c *channels) del(channel *Channel) {
	c.Lock()
	defer c.Unlock()
	delete(c.m_addr, channel.a)
	delete(c.m_chan, channel.c)
	c.trudp.ChannelDel(channel.c)
	c.teo.log.Println("client disconnec:", channel.a)
}

// get channel by address or by trudp channel
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

// new create new teonet channel
func (c *channels) new( /* address string,  */ channel *trudp.Channel) *Channel {
	address := newChannelPrefix + trudp.RandomString(addressLen-len(newChannelPrefix))
	return &Channel{address, channel}
}

// Channel get teonet channel by address
func (teo Teonet) Channel(addr string) (ch *Channel, exists bool) {
	return teo.channels.get(addr)
}

type Channel struct {
	a string         // Teonet address
	c *trudp.Channel // Trudp channel
}

func (c Channel) ServerMode() bool {
	return c.c.ServerMode()
}

func (c Channel) Triptime() time.Duration {
	return c.c.Triptime()
}

func (c Channel) Send(data []byte) (id uint32, err error) {
	return c.c.Send(data)
}

func (c Channel) SendAnswer(data []byte) (id uint32, err error) {
	return c.c.SendAnswer(data)
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

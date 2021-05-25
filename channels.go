package teonet

import (
	"sync"
	"time"

	"github.com/kirill-scherba/trudp"
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
		c.del(channel)
		if ch.c.String() != channel.c.String() {
			c.trudp.ChannelsDel(ch.c)
		}
	}

	// add new channel
	c.Lock()
	defer c.Unlock()
	c.m_addr[channel.a] = channel
	c.m_chan[channel.c] = channel
}

// del channel by address
func (c *channels) del(channel *Channel) {
	c.Lock()
	defer c.Unlock()
	delete(c.m_addr, channel.a)
	delete(c.m_chan, channel.c)
	c.teo.log.Println("client disconnec:", channel.a)
}

// get channel by address or by trudp channel
func (c *channels) get(attr interface{}) (ch *Channel, exsists bool) {
	c.RLock()
	defer c.RUnlock()
	switch v := attr.(type) {
	case string:
		ch, exsists = c.m_addr[v]
	case *trudp.Channel:
		ch, exsists = c.m_chan[v]
	}
	return
}

// new create new teonet channel
func (c *channels) new(address string, channel *trudp.Channel) *Channel {
	return &Channel{address, channel}
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

func (c Channel) SendAnswer(data []byte) (id uint32, err error) {
	return c.c.SendAnswer(data)
}

func (c Channel) String() string {
	if c.a == "" {
		return c.c.String()
	}
	return c.a
}

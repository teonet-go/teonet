// Copyright 2021 Kirill Scherba <kirill@scherba.ru>. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Teonet v4

package teonet

import (
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/kirill-scherba/teonet-go/teolog/teolog"
	"github.com/kirill-scherba/trudp"
)

const Version = "0.2.18"

// nMODULEteo is current module name
var nMODULEteo = "Teonet"

// Logo print teonet logo
func Logo(title, ver string) {
	fmt.Println(LogoString(title, ver))
}

func LogoString(title, ver string) string {
	return fmt.Sprint("" +
		" _____                     _   \n" +
		"|_   _|__  ___  _ __   ___| |_  v4\n" +
		"  | |/ _ \\/ _ \\| '_ \\ / _ \\ __|\n" +
		"  | |  __/ (_) | | | |  __/ |_ \n" +
		"  |_|\\___|\\___/|_| |_|\\___|\\__|\n" +
		"\n" +
		title + " ver " + ver + ", based on Teonet v4 ver " + Version +
		"\n",
	)
}

// Log get teonet log to use it in application and inside teonet
func Log() *log.Logger {
	return log.New(os.Stdout, "", log.Ltime|log.Lmicroseconds /* |log.Lshortfile */)
}

// reader is Main teonet reader
func reader(teo *Teonet, c *Channel, p *Packet, e *Event) {

	// Delete channel on err after all other reader process this error
	defer func() {
		if e.Err != nil {
			teo.channels.del(c)
		}
	}()

	// Check error and 'connect to peer connected' processing
	// if e.Err == nil && (teo.connectToConnectedPeer(c, p) || teo.connectToConnectedClient(c, p)) {
	if e.Event == EventData && (teo.connectToConnectedPeer(c, p) || teo.connectToConnectedClient(c, p)) {
		return
	}

	// Send to subscribers readers (to readers from teo.subscribe)
	if teo.subscribers.send(teo, c, p, e) {
		return
	}

	// Send to client readers (to reader from teonet.Init)
	for i := range teo.clientReaders {
		if teo.clientReaders[i] != nil {
			if teo.clientReaders[i](teo, c, p, e) {
				break
			}
		}
	}
}

type LogFilterT = trudp.LogFilterT

// New create new teonet connection. The attr parameters:
//   int - port number to teonet listen
//   string - internal log Level to show teonet debug messages
//   bool - set true to show trudp statistic table
//   *log.Logger - common logger to show messages in application and teonet, may be created with teonet.Log() function
//   func(c *Channel, p *Packet, e *Event) - message receiver
func New(appName string, attr ...interface{}) (teo *Teonet, err error) {

	// Parse attributes
	var param struct {
		port        int
		showTrudp   bool
		logLevel    string
		logFilter   LogFilterT
		log         *log.Logger
		reader      Treceivecb
		api         ApiInterface
		configFiles ConfigFiles
	}
	for i := range attr {
		switch d := attr[i].(type) {
		// Local port
		case int:
			param.port = d
		// Log level
		case string:
			param.logLevel = d
		// Log filter
		case trudp.LogFilterT:
			param.logFilter = d
		// Show trudp flag
		case bool:
			param.showTrudp = d
		// Logger
		case *log.Logger:
			param.log = d
		// Treceivecb:
		case func(teo *Teonet, c *Channel, p *Packet, e *Event) bool:
			param.reader = d
		// TreceivecbShort:
		case func(c *Channel, p *Packet, e *Event) bool:
			param.reader = func(t *Teonet, c *Channel, p *Packet, e *Event) bool {
				return d(c, p, e)
			}
		// API interface
		case ApiInterface:
			param.api = d
		// Config file folder
		case ConfigFiles:
			param.configFiles = d
		// Some enother (incorrect) attribute
		default:
			err = fmt.Errorf("incorrect attribute type %T", d)
			return
		}
	}
	if param.logLevel == "" {
		param.logLevel = "NONE"
	}
	if param.log == nil {
		param.log = Log()
	}
	log := param.log

	// Create new teonet holder
	teo = new(Teonet)
	teo.newSubscribers()
	teo.newPeerRequests()
	teo.newConnRequests()
	teo.log = log

	// Create config holder and read config
	err = teo.newConfig(appName, log, param.configFiles)
	if err != nil {
		return
	}

	// Add client readers
	teo.addApiReader(param.api)
	teo.addClientReader(param.reader)

	// Init trudp and start listen port to get messages
	teo.trudp, err = trudp.Init(param.port, teo.config.trudpPrivateKey, teo.log,
		param.logLevel, trudp.LogFilterT(param.logFilter),

		// Receive data callback
		func(c *trudp.Channel, p *trudp.Packet, err error) {
			ch, ok := teo.channels.get(c)
			if !ok {
				if teo.auth != nil && c == teo.auth.c {
					ch = teo.auth
				} else {
					ch = teo.channels.new(c)
				}
			}

			// Create packet
			var pac *Packet
			if p != nil {
				pac = &Packet{p, ch.a, false}
			}

			// Create Disconnect, TeonetDisconnect or Data Events
			e := new(Event)
			if err != nil {
				e.Err = err
				if ch == teo.auth {
					e.Event = EventTeonetDisconnected
				} else {
					e.Event = EventDisconnected
				}
			} else {
				e.Event = EventData
			}

			// Send packet and event to main teonet reader
			reader(teo, ch, pac, e)
		},

		// Connect to this server callback
		func(c *trudp.Channel, err error) {
			// Wait this trudp channel connected to teonet channel and delete
			// it if not connected during timeout
			_, exists := teo.channels.get(c)
			if exists {
				return
			}
			go func(c *trudp.Channel) {
				time.Sleep(trudp.ClientConnectTimeout)
				ch, exists := teo.channels.get(c)
				if !exists {
					if newch, ok := teo.channels.getByIP(c.String()); !ok {
						teolog.Log(teolog.DEBUG, nMODULEteo, "remove dummy trudp channel:", c, ch)
						c.ChannelDel(c)
					} else {
						teolog.Log(teolog.DEBUGvv, nMODULEteo, "trudp channel was reconnected:", c.String(), newch)
					}
				} else if ch.IsNew() {
					teolog.Log(teolog.DEBUG, nMODULEteo, "remove dummy(new) teonet channel:", c, ch)
					teo.channels.del(ch)
				}
			}(c)
		},
	)
	if err != nil {
		teolog.Log(teolog.ERROR, nMODULEteo, "can't initial trudp, error:", err)
		return
	}
	teo.newChannels()
	teo.newPuncher()
	teolog.Log(teolog.CONNECT, nMODULEteo, "start listen teonet at port", teo.trudp.Port())

	if param.showTrudp {
		teo.ShowTrudp(true)
	}

	return
}

type Teonet struct {
	config        *config
	trudp         *trudp.Trudp
	log           *log.Logger
	clientReaders []Treceivecb
	subscribers   *subscribers
	channels      *channels
	auth          *Channel
	peerRequests  *connectRequests
	connRequests  *connectRequests
	puncher       *puncher
}

type Treceivecb func(teo *Teonet, c *Channel, p *Packet, e *Event) bool
type TreceivecbShort func(c *Channel, p *Packet, e *Event) bool

type Event struct {
	Event TeonetEventType
	Err   error
}

type TeonetEventType byte

// Teonet events
const (
	EventNone TeonetEventType = iota

	// Event when Teonet client initialized and start listen, Err = nil
	EventTeonetInit

	// Event when Connect to teonet r-host, Err = nil
	EventTeonetConnected

	// Event when Disconnect from teonet r-host, Err = dosconnect error
	EventTeonetDisconnected

	// Event when Connect to peer, Err = nil
	EventConnected

	// Event when Disconnect from peer, Err = dosconnect error
	EventDisconnected

	// Event when Data Received, Err = nil
	EventData
)

func (e Event) String() (str string) {
	switch e.Event {
	case EventNone:
		str = "EventNone"
	case EventTeonetInit:
		str = "EventTeonetInit"
	case EventTeonetConnected:
		str = "EventTeonetConnected"
	case EventTeonetDisconnected:
		str = "EventTeonetDisconnected"
	case EventConnected:
		str = "EventConnected"
	case EventDisconnected:
		str = "EventDisconnected"
	case EventData:
		str = "EventData"
	}
	return
}

func (teo Teonet) Rhost() *Channel { return teo.auth }

// addClientReader add teonet client reader
func (teo *Teonet) addClientReader(reader Treceivecb) {
	teo.clientReaders = append(teo.clientReaders, reader)
}

// AddReader add teonet client reader
func (teo *Teonet) AddReader(reader TreceivecbShort) {
	teo.clientReaders = append(teo.clientReaders,
		func(teo *Teonet, c *Channel, p *Packet, e *Event) bool {
			return reader(c, p, e)
		})
}

// ShowTrudp show/stop trudp statistic
func (teo Teonet) ShowTrudp(set bool) {
	teo.trudp.SetShowStat(set)
}

var ErrPeerNotConnected = errors.New("peer does not connected")

// Send data to peer
func (teo *Teonet) SendTo(addr string, data []byte, attr ...interface{}) (id uint32, err error) {
	// Check address
	c, ok := teo.channels.get(addr)
	if !ok {
		err = ErrPeerNotConnected
		return
	}
	// Add teo to attr, it need for subscribe to answer
	if len(attr) > 0 {
		attr = append([]interface{}{teo}, attr...)
	}
	// Send to channel
	return c.Send(data, attr...)
}

// Connected return true if peer with selected address is connected now
func (teo *Teonet) Connected(addr string) (ok bool) {
	_, ok = teo.channels.get(addr)
	return
}

// Log get teonet log
func (teo Teonet) Log() *log.Logger {
	return teo.log
}

// Port get teonet local port
func (teo Teonet) Port() uint32 {
	return uint32(teo.trudp.Port())
}

// Get this app Address
func (teo Teonet) MyAddr() string {
	return teo.config.Address
}
